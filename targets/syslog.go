//go:build !windows && !nacl && !plan9

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
	params *SyslogOptions
	writer *syslog.Writer
}

// SyslogOptions provides parameters for dialing a syslog daemon.
type SyslogOptions struct {
	IP       string `json:"ip,omitempty"` // deprecated (use Host instead)
	Host     string `json:"host"`
	Port     int    `json:"port"`
	TLS      bool   `json:"tls"`
	Cert     string `json:"cert"`
	Insecure bool   `json:"insecure"`
	Tag      string `json:"tag"`
}

// GetHost returns the host to connect to, using the Host field if set,
// otherwise falling back to the deprecated IP field. An empty result means
// local syslog.
func (so SyslogOptions) GetHost() string {
	return hostOrIP(so.Host, so.IP)
}

// CheckValid returns an error if these options are not valid.
//
// Leaving both the host and the port unset selects local syslog, which `Init`
// connects to with an empty address. Setting either one requires both, since a
// remote daemon cannot be reached without them.
func (so SyslogOptions) CheckValid() error {
	host := so.GetHost()
	if host != "" && so.Port == 0 {
		return errors.New("missing port")
	}
	if host == "" && so.Port != 0 {
		return errors.New("missing host")
	}
	if so.Port < 0 || so.Port > 65535 {
		return fmt.Errorf("port is invalid (%d)", so.Port)
	}
	if err := logr.CheckOptionText("host", host, logr.MaxHostnameLen); err != nil {
		return err
	}
	if err := logr.CheckOptionText("tag", so.Tag, logr.MaxTagLen); err != nil {
		return err
	}
	return logr.CheckOptionLen("cert", so.Cert, logr.MaxCertLen)
}

// NewSyslogTarget creates a target capable of outputting log records to remote or local syslog, with or without TLS.
func NewSyslogTarget(params *SyslogOptions) (*Syslog, error) {
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

	host := s.params.GetHost()

	if s.params.TLS {
		network = "tcp+tls"
		config = &tls.Config{InsecureSkipVerify: s.params.Insecure}

		pool, err := GetCertPoolOrNil(s.params.Cert)
		if err != nil {
			return err
		}
		if pool != nil {
			config.RootCAs = pool
		}
	}
	raddr := fmt.Sprintf("%s:%d", host, s.params.Port)
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
