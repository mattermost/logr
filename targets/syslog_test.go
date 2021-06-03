// +build !windows,!nacl,!plan9

package targets_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleSyslog() {
	lgr, _ := logr.New()
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	params := &targets.SyslogOptions{
		IP:   "localhost",
		Port: 514,
		Tag:  "logrtest",
	}
	t, err := targets.NewSyslogTarget(params)
	if err != nil {
		panic(err)
	}
	err = lgr.AddTarget(t, "syslogTest", filter, formatter, 1000)
	if err != nil {
		panic(err)
	}

	logger := lgr.NewLogger().With(logr.String("name", "wiggin")).Sugar()

	logger.Errorf("the erroneous data is %s", test.StringRnd(10))
	logger.Warnf("strange data: %s", test.StringRnd(5))
	logger.Debug("XXX")
	logger.Trace("XXX")

	err = lgr.Shutdown()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func TestSyslogPlain(t *testing.T) {
	plain := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	syslogger(t, plain)
}

func syslogger(t *testing.T, formatter logr.Formatter) {
	opt := logr.OnLoggerError(func(err error) {
		t.Error(err)
	})
	lgr, err := logr.New(opt)
	require.NoError(t, err)

	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Panic}
	params := &targets.SyslogOptions{
		Tag: "logrtest",
	}
	target, err := targets.NewSyslogTarget(params)
	require.NoError(t, err)

	err = lgr.AddTarget(target, "syslogTest2", filter, formatter, 1000)
	require.NoError(t, err)

	cfg := test.DoSomeLoggingCfg{
		Lgr:        lgr,
		Goroutines: 3,
		Loops:      5,
		GoodToken:  "Woot!",
		BadToken:   "XXX!!XXX",
		Lvl:        logr.Warn,
		Delay:      time.Millisecond * 1,
	}
	test.DoSomeLogging(cfg)

	err = lgr.Shutdown()
	require.NoError(t, err)
}

func Test_getCertPool(t *testing.T) {
	tests := []struct {
		name    string
		cert    string
		wantErr bool
	}{
		{name: "garbage in, garbage out", wantErr: true, cert: "THISISNOTACERT"},
		{name: "good cert base64", wantErr: false, cert: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURqekNDQW5lZ0F3SUJBZ0lSQVBZZlJTd2R6S29wQkt4WXhLcXNsSlV3RFFZSktvWklodmNOQVFFTEJRQXcKSnpFbE1DTUdBMVVFQXd3Y1RXRjBkR1Z5Ylc5emRDd2dTVzVqTGlCSmJuUmxjbTVoYkNCRFFUQWVGdzB4T1RBegpNakl3TURFME1UVmFGdzB5TWpBek1EWXdNREUwTVRWYU1Ec3hPVEEzQmdOVkJBTVRNRTFoZEhSbGNtMXZjM1FzCklFbHVZeTRnU1c1MFpYSnVZV3dnU1c1MFpYSnRaV1JwWVhSbElFRjFkR2h2Y21sMGVUQ0NBU0l3RFFZSktvWkkKaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFNamxpUmRtdm5OTDR1L0pyL00yZFB3UW1USlhFQlkvVnE5UQp2QVU1MlgzdFJNQ1B4Y2FGeit4NmZ0dXZkTzJOZG9oWEdBbXR4OVFVNUxaY3ZGZVREcG9WRUJvOUErNGp0THZECkRaWWFUTkxwSm1vU29KSGFEYmRXWCtPQU9xeURpV1M3NDFMdWlNS1dIaGV3OVFPaXNhdDJaSU5QeGptQWQ5d0UKeHRoVE1nenN2N01VcW5NZXI4VTVPR1EwUXk3d0FtTlJjKzJLM3FQd2t4ZTJSVXZjdGU1MERVRk5neEVnaW5zaAp2cmtPWFIzODN2VUNaZnU3MnF1OG9nZ2ppUXB5VGxsdTVqZTJBcDZKTGpZTGtFTWlNcXJZQUR1V29yL1pId2E2CldyRnFWRVR4V2ZBVjV1OUVoMHdaTS9LS1l3UlF1dzl5K05hbnM3N0ZtVWwxdFZXV05OOENBd0VBQWFPQm9UQ0IKbmpBTUJnTlZIUk1FQlRBREFRSC9NQjBHQTFVZERnUVdCQlFZNFVxc3d5cjJoTy9IZXRadDJSRHhKZFRJUGpCaQpCZ05WSFNNRVd6QlpnQlJGWlhWZzJaNXROSXNXZVdqQkxFeTJ5ektiTUtFcnBDa3dKekVsTUNNR0ExVUVBd3djClRXRjBkR1Z5Ylc5emRDd2dTVzVqTGlCSmJuUmxjbTVoYkNCRFFZSVVFaWZHVU9NK2JJRlpvMXRralpCNVlHQnIKMHhFd0N3WURWUjBQQkFRREFnRUdNQTBHQ1NxR1NJYjNEUUVCQ3dVQUE0SUJBUUFFZGV4TDMwUTB6QkhtUEFIOApMaGRLN2RielcxQ21JTGJ4UlpsS0F3Uk4raEtSWGlNVzNNSElraE51b1Y5QWV2NjAyUStqYTRsV3NSaS9rdE9MCm5pMUZXeDVnU1NjZ2RHOEpHajQ3ZE9tb1QzdlhLWDcrdW1pdjRyUUxQRGw5L0RLTXV2MjA0T1lKcTZWVCt1TlUKNkM2a0wxNTdqR0pFTzc2SDRmTVo4b1lzRDdTcTB6amlOS3R1Q1lpaTBuZ0gzajNnQjFqQUNMcVJndmVVN01kVApwcU9WMktmWTMxK2g4VkJ0a1V2bGpOenRROXhOWThGam10MFNNZjdFM0ZhVWNhYXIzWkNyNzBHNWFVM2RLYmU3CjQ3dkdPQmE1dENxdzRZSzBqZ0RLaWQzSUpRdWw5YTNKMW1Tc0g4V3kzdG85Y0FWNEtHWkJRTG56Q1gxNWEvK3YKM3lWaAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tIAotLS0tLUJFR0lOIENFUlRJRklDQVRFLS0tLS0KTUlJRGZqQ0NBbWFnQXdJQkFnSVVFaWZHVU9NK2JJRlpvMXRralpCNVlHQnIweEV3RFFZSktvWklodmNOQVFFTApCUUF3SnpFbE1DTUdBMVVFQXd3Y1RXRjBkR1Z5Ylc5emRDd2dTVzVqTGlCSmJuUmxjbTVoYkNCRFFUQWVGdzB4Ck9UQXpNakV5TVRJNE5ETmFGdzB5T1RBek1UZ3lNVEk0TkROYU1DY3hKVEFqQmdOVkJBTU1IRTFoZEhSbGNtMXYKYzNRc0lFbHVZeTRnU1c1MFpYSnVZV3dnUTBFd2dnRWlNQTBHQ1NxR1NJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFSwpBb0lCQVFESDBYcTVyTUJHcEtPVldUcGI1TW5hSklXRlAvdk90dkVrKzdoVnJmT2ZlMS81eDBLazNVZ0FIajg1Cm90YUVaRDFMaG4vSkxrRXFDaUUvVVhNSkZ3SkRsTmNPNENrZEtCU3BZWDRiS0FxeTVxL1gzUXdpb01TTnBKRzEKK1lZck5HQkgwc2dLY0tqeUNhTGhtcVlMRDB4WkRWT21XSVlCVTlqVVB5WHc1VTB0bnNWclRxR014VmttMXhDWQprckNXTjFab1VyTHZMME1DWmM1cXB4b1BUb3ByOVVPOWNxU0JTdXk2QlZXVnVFV0JaaHBxSHQrdWw4VnhoenpZCnExazRsN3IycXcrL3dtMWlKQmVkVGVCVmVXTmFnOEphVmZMZ3UrL1c3b0pWbFBPMzJQbzdwbnZIcDhpSjNiNEsKelh5VkhhVFg0UzZFbSs2TFY4ODU1VFlyU2h6bEFnTUJBQUdqZ2FFd2daNHdIUVlEVlIwT0JCWUVGRVZsZFdEWgpubTAwaXhaNWFNRXNUTGJMTXBzd01HSUdBMVVkSXdSYk1GbUFGRVZsZFdEWm5tMDBpeFo1YU1Fc1RMYkxNcHN3Cm9TdWtLVEFuTVNVd0l3WURWUVFEREJ4TllYUjBaWEp0YjNOMExDQkpibU11SUVsdWRHVnlibUZzSUVOQmdoUVMKSjhaUTR6NXNnVm1qVzJTTmtIbGdZR3ZURVRBTUJnTlZIUk1FQlRBREFRSC9NQXNHQTFVZER3UUVBd0lCQmpBTgpCZ2txaGtpRzl3MEJBUXNGQUFPQ0FRRUFQaUNXRm1vcHlBa1kyVDNaeW80eWFSUGhYMStWT1RNS0p0WTZFVWhxCi9HSHo2a3pFeXZDVUJmME44OTJjaWJHeGVrckVvSXRZOU5xTzZSUVJmb3dnK0duNWtjMTN6NE55TDJXOC9lb1QKWHkwWnZmYVFiVSsrZlE2cFZ0V3RNYmxETVU5eGlZZDcvTUR2SnBPMzI4bDFWaGNkcDhrRWkrbEN2cHkwc0NSYwpQeHpQaGJnQ01BYlpFR3grNFRNUWQ0U1pLemxSeFcvMmZmbHBSZWg2djFEdjBWRFVTWVFXd3NVbmFMcGRLSGZoCmE1azB2dXlTWWNzekU0WUtsWTB6YWtlRmxKZnA3ZkJwMXhUd2NkVzhhVGZ3MTVFaWNQTXdUYzZ4eEE0SkpVSngKY2RkdTgxN24xbmF5SzV1NnI5UWgxb0lWa3IwbkM5WUVMTU15NGRwUGdKODhTQT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"},
		{name: "good cert file", wantErr: false, cert: "test-tls-client-cert.pem"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := targets.GetCertPool(tt.cert)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, pool)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pool)

				// Test PEM has 2 certs.
				subjects := pool.Subjects()
				assert.Len(t, subjects, 2)
			}
		})
	}
}
