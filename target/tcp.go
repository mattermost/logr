// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package target

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/mattermost/logr"
	"github.com/wiggin77/merror"
)

const (
	DialTimeoutSecs             = 30
	WriteTimeoutSecs            = 30
	RetryBackoffMillis    int64 = 100
	MaxRetryBackoffMillis int64 = 30 * 1000 // 30 seconds
)

// Tcp outputs log records to raw socket server.
type Tcp struct {
	logr.Basic

	params *TcpParams
	addy   string

	mutex    sync.Mutex
	conn     net.Conn
	monitor  chan struct{}
	shutdown chan struct{}
}

// TcpParams provides parameters for dialing a socket server.
type TcpParams struct {
	IP       string `json:"IP"`
	Port     int    `json:"Port"`
	TLS      bool   `json:"TLS"`
	Cert     string `json:"Cert"`
	Insecure bool   `json:"Insecure"`
}

// NewTcpTarget creates a target capable of outputting log records to a raw socket, with or without TLS.
func NewTcpTarget(filter logr.Filter, formatter logr.Formatter, params *TcpParams, maxQueue int) (*Tcp, error) {
	tcp := &Tcp{
		params:   params,
		addy:     fmt.Sprintf("%s:%d", params.IP, params.Port),
		monitor:  make(chan struct{}),
		shutdown: make(chan struct{}),
	}
	tcp.Basic.Start(tcp, tcp, filter, formatter, maxQueue)

	return tcp, nil
}

// getConn provides a net.Conn.  If a connection already exists, it is returned immediately,
// otherwise this method blocks until a new connection is created, timeout or shutdown.
func (tcp *Tcp) getConn(reporter func(err interface{})) (net.Conn, error) {
	tcp.mutex.Lock()
	defer tcp.mutex.Unlock()

	if tcp.conn != nil {
		return tcp.conn, nil
	}

	type result struct {
		conn net.Conn
		err  error
	}

	connChan := make(chan result)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*DialTimeoutSecs)
	defer cancel()

	go func(ctx context.Context, ch chan result) {
		conn, err := tcp.dial(ctx)
		if err != nil {
			reporter(fmt.Errorf("log target %s connection error: %w", tcp.String(), err))
			return
		}
		tcp.conn = conn
		tcp.monitor = make(chan struct{})
		go monitor(tcp.conn, tcp.monitor)
		ch <- result{conn: conn, err: err}
	}(ctx, connChan)

	select {
	case <-tcp.shutdown:
		return nil, errors.New("shutdown")
	case res := <-connChan:
		return res.conn, res.err
	}
}

// dial connects to a TCP socket, and optionally performs a TLS handshake.
// A non-nil context must be provided which can cancel the dial.
func (tcp *Tcp) dial(ctx context.Context) (net.Conn, error) {
	var dialer net.Dialer
	dialer.Timeout = time.Second * DialTimeoutSecs
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", tcp.params.IP, tcp.params.Port))
	if err != nil {
		return nil, err
	}

	if !tcp.params.TLS {
		return conn, nil
	}

	tlsconfig := &tls.Config{
		ServerName:         tcp.params.IP,
		InsecureSkipVerify: tcp.params.Insecure,
	}
	if tcp.params.Cert != "" {
		pool, err := getCertPool(tcp.params.Cert)
		if err != nil {
			return nil, err
		}
		tlsconfig.RootCAs = pool
	}

	tlsConn := tls.Client(conn, tlsconfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}
	return tlsConn, nil
}

func (tcp *Tcp) close() error {
	tcp.mutex.Lock()
	defer tcp.mutex.Unlock()

	var err error
	if tcp.conn != nil {
		close(tcp.monitor)
		err = tcp.conn.Close()
		tcp.conn = nil
	}
	return err
}

// Shutdown stops processing log records after making best effort to flush queue.
func (tcp *Tcp) Shutdown(ctx context.Context) error {
	merr := merror.New()

	if err := tcp.Basic.Shutdown(ctx); err != nil {
		merr.Append(err)
	}

	if err := tcp.close(); err != nil {
		merr.Append(err)
	}

	close(tcp.shutdown)
	return merr.ErrorOrNil()
}

// Write converts the log record to bytes, via the Formatter, and outputs to the socket.
// Called by dedicated target goroutine and will block until success or shutdown.
func (tcp *Tcp) Write(rec *logr.LogRec) error {
	_, stacktrace := tcp.IsLevelEnabled(rec.Level())

	buf := rec.Logger().Logr().BorrowBuffer()
	defer rec.Logger().Logr().ReleaseBuffer(buf)

	buf, err := tcp.Formatter().Format(rec, stacktrace, buf)
	if err != nil {
		return err
	}

	try := 1
	backoff := RetryBackoffMillis
	for {
		select {
		case <-tcp.shutdown:
			return err
		default:
		}

		reporter := rec.Logger().Logr().ReportError

		conn, err := tcp.getConn(reporter)
		if err != nil {
			reporter(fmt.Errorf("log target %s connection error: %w", tcp.String(), err))
			backoff = tcp.sleep(backoff)
			continue
		}

		err = conn.SetWriteDeadline(time.Now().Add(time.Second * WriteTimeoutSecs))
		if err != nil {
			reporter(fmt.Errorf("log target %s set write deadline error: %w", tcp.String(), err))
		}

		_, err = buf.WriteTo(conn)
		if err == nil {
			return nil
		}

		reporter(fmt.Errorf("log target %s write error: %w", tcp.String(), err))

		_ = tcp.close()

		backoff = tcp.sleep(backoff)
		try++
	}
}

// monitor continuously tries to read from the connection to detect socket close.
// This is needed because TCP target uses a write only socket and Linux systems
// take a long time to detect a loss of connectivity on a socket when only writing;
// the writes simply fail without an error returned.
func monitor(conn net.Conn, done <-chan struct{}) {
	buf := make([]byte, 1)
	for {
		select {
		case <-done:
			return
		case <-time.After(1 * time.Second):
		}

		err := conn.SetReadDeadline(time.Now().Add(time.Second * 30))
		if err != nil {
			continue
		}

		_, err = conn.Read(buf)

		if errt, ok := err.(net.Error); ok && errt.Timeout() {
			// read timeout is expected, keep looping.
			continue
		}

		// Any other error closes the connection, forcing a reconnect.
		conn.Close()
		return
	}
}

// String returns a string representation of this target.
func (tcp *Tcp) String() string {
	return fmt.Sprintf("TcpTarget[%s:%d]", tcp.params.IP, tcp.params.Port)
}

func (tcp *Tcp) sleep(backoff int64) int64 {
	select {
	case <-tcp.shutdown:
	case <-time.After(time.Millisecond * time.Duration(backoff)):
	}

	nextBackoff := backoff + (backoff >> 1)
	if nextBackoff > MaxRetryBackoffMillis {
		nextBackoff = MaxRetryBackoffMillis
	}
	return nextBackoff
}

// getCertPool returns a x509.CertPool containing the cert(s)
// from `cert`, which can be a path to a .pem or .crt file,
// or a base64 encoded cert.
func getCertPool(cert string) (*x509.CertPool, error) {
	if cert == "" {
		return nil, errors.New("no cert provided")
	}

	// first treat as a file and try to read.
	serverCert, err := ioutil.ReadFile(cert)
	if err != nil {
		// maybe it's a base64 encoded cert
		serverCert, err = base64.StdEncoding.DecodeString(cert)
		if err != nil {
			return nil, errors.New("cert cannot be read")
		}
	}

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(serverCert); ok {
		return pool, nil
	}
	return nil, errors.New("cannot parse cert")
}
