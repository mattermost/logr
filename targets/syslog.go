// +build !windows,!nacl,!plan9

package targets

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/mattermost/logr/v2"
	syslog "github.com/wiggin77/srslog"
)

// Syslog outputs log records to local or remote syslog.
type Syslog struct {
	params *SyslogParams
	writer *syslog.Writer
}

// SyslogParams provides parameters for dialing a syslog daemon.
type SyslogParams struct {
	IP       string
	Port     int
	Tag      string
	TLS      bool
	Cert     string
	Insecure bool
}

// NewSyslogTarget creates a target capable of outputting log records to remote or local syslog, with or without TLS.
func NewSyslogTarget(params *SyslogParams) (*Syslog, error) {
	if params == nil {
		return nil, errors.New("params cannot be nil")
	}

	s := &Syslog{
		params: params,
	}
	return s, nil
}

// Init is called once to initialize the target.
func (s *Syslog) Init() error {
	network := "tcp"
	var config *tls.Config

	if s.params.TLS {
		network = "tcp+tls"
		config = &tls.Config{InsecureSkipVerify: s.params.Insecure}
		if s.params.Cert != "" {
			pool, err := GetCertPool(s.params.Cert)
			if err != nil {
				return err
			}
			config.RootCAs = pool
		}
	}
	raddr := fmt.Sprintf("%s:%d", s.params.IP, s.params.Port)
	if raddr == ":0" {
		// If no IP:port provided then connect to local syslog.
		raddr = ""
		network = ""
	}

	var err error
	s.writer, err = syslog.DialWithTLSConfig(network, raddr, syslog.LOG_INFO, s.params.Tag, config)
	return err
}

// Write outputs bytes to this file target.
func (s *Syslog) Write(p []byte, rec *logr.LogRec) (int, error) {
	txt := string(p)
	n := len(txt)
	var err error

	switch rec.Level() {
	case logr.Panic, logr.Fatal:
		err = s.writer.Crit(txt)
	case logr.Error:
		err = s.writer.Err(txt)
	case logr.Warn:
		err = s.writer.Warning(txt)
	case logr.Debug, logr.Trace:
		err = s.writer.Debug(txt)
	default:
		// logr.Info plus all custom levels.
		err = s.writer.Info(txt)
	}

	if err != nil {
		n = 0
		// syslog writer will try to reconnect.
	}
	return n, err
}

// Shutdown is called once to free/close any resources.
// Target queue is already drained when this is called.
func (s *Syslog) Shutdown() error {
	return s.writer.Close()
}
