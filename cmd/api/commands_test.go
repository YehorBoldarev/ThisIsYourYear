package main

import (
	"bytes"
	"net"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCommand_UsageErrors(t *testing.T) {
	cases := map[string][]string{
		"unknown command":          {"bogus"},
		"admin without subcommand": {"admin"},
		"unknown admin command":    {"admin", "drop-database"},
		"reset without email":      {"admin", "reset-password"},
		"reset with stray arg":     {"admin", "reset-password", "--email", "a@b.co", "extra"},
		"list-users with args":     {"admin", "list-users", "now"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			require.Equal(t, 2, runCommand(args, &stdout, &stderr))
			require.Contains(t, stderr.String(), "Usage:")
		})
	}
}

func TestRunCommand_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, runCommand([]string{"help"}, &stdout, &stderr))
	require.Contains(t, stdout.String(), "reset-password")
}

func TestHealthcheck(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var status atomic.Int32
	status.Store(http.StatusOK)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(int(status.Load()))
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	t.Setenv("HTTP_PORT", port)

	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, runCommand([]string{"healthcheck"}, &stdout, &stderr))
	require.Equal(t, "ok\n", stdout.String())

	status.Store(http.StatusServiceUnavailable)
	require.Equal(t, 1, runCommand([]string{"healthcheck"}, &stdout, &stderr))
	require.Contains(t, stderr.String(), "readyz returned 503")
}
