package test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wiggin77/merror"
)

// SocketServer is a simple socket server used for testing TCP log targets.
// Note: There is more synchronization here than normally needed to avoid flaky tests.
//
//	For example, it's possible for a unit test to create a SocketServer, attempt
//	writing to it, and stop the socket server before "go ss.listen()" gets scheduled.
type SocketServer struct {
	listener      net.Listener
	anyConn       chan struct{}
	anyConnClosed int32
	buf           *Buffer
	conns         map[string]*socketServerConn
	mux           sync.Mutex
}

type socketServerConn struct {
	raddy string
	conn  net.Conn
	done  chan struct{}
}

func NewSocketServer(port int, buf *Buffer) (*SocketServer, error) {
	ss := &SocketServer{
		buf:     buf,
		conns:   make(map[string]*socketServerConn),
		anyConn: make(chan struct{}),
	}

	addy := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp4", addy)
	if err != nil {
		return nil, err
	}
	ss.listener = l

	go ss.listen()
	return ss, nil
}

func (ss *SocketServer) listen() {
	for {
		conn, err := ss.listener.Accept()
		if err != nil {
			return
		}
		sconn := &socketServerConn{raddy: conn.RemoteAddr().String(), conn: conn, done: make(chan struct{})}
		ss.registerConnection(sconn)
		go ss.handleConnection(sconn)
	}
}

func (ss *SocketServer) WaitForAnyConnection() error {
	var err error
	select {
	case <-ss.anyConn:
	case <-time.After(15 * time.Second):
		err = errors.New("wait for any connection timed out")
	}
	return err
}

func (ss *SocketServer) handleConnection(sconn *socketServerConn) {
	if atomic.CompareAndSwapInt32(&ss.anyConnClosed, 0, 1) {
		close(ss.anyConn)
	}

	defer ss.unregisterConnection(sconn)
	buf := make([]byte, 1024)

	for {
		n, err := sconn.conn.Read(buf)
		if n > 0 {
			_, _ = ss.buf.Write(buf[:n])
		}
		if err == io.EOF {
			ss.signalDone(sconn)
			return
		}
	}
}

func (ss *SocketServer) registerConnection(sconn *socketServerConn) {
	ss.mux.Lock()
	defer ss.mux.Unlock()
	ss.conns[sconn.raddy] = sconn
}

func (ss *SocketServer) unregisterConnection(sconn *socketServerConn) {
	ss.mux.Lock()
	defer ss.mux.Unlock()
	delete(ss.conns, sconn.raddy)
}

func (ss *SocketServer) signalDone(sconn *socketServerConn) {
	ss.mux.Lock()
	defer ss.mux.Unlock()
	close(sconn.done)
}

func (ss *SocketServer) StopServer(wait bool) error {
	errs := merror.New()
	ss.listener.Close()

	ss.mux.Lock()
	// defensive copy; no more connections can be accepted so copy will stay current.
	conns := make(map[string]*socketServerConn, len(ss.conns))
	for k, v := range ss.conns {
		conns[k] = v
	}
	ss.mux.Unlock()

	for _, sconn := range conns {
		if err := sconn.conn.Close(); err != nil {
			errs.Append(err)
			continue
		}

		if !wait {
			continue
		}

		select {
		case <-sconn.done:
		case <-time.After(time.Second * 5):
			errs.Append(errors.New("timed out"))
		}
	}
	return errs.ErrorOrNil()
}
