package http

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestServer_ServesAndShutsDownGracefully starts the real Server, confirms it
// serves a request, then cancels its context and asserts it shuts down cleanly
// (Run returns nil) within the deadline.
func TestServer_ServesAndShutsDownGracefully(t *testing.T) {
	port := freePort(t)

	router := NewRouter(RouterDeps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Pinger: stubPinger{nil},
	})
	srv := NewServer(ServerConfig{
		Port:            port,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: 5 * time.Second,
	}, router, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- srv.Run(ctx) }()

	// Wait until the server is accepting connections, then verify it serves.
	base := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	requireServing(t, base+"/healthz")

	// Trigger graceful shutdown.
	cancel()

	select {
	case err := <-runErr:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("server did not shut down within deadline")
	}
}

// freePort reserves an available TCP port and releases it for the server to bind.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

// requireServing polls the URL until it responds 200 or the deadline elapses.
func requireServing(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server never served 200 at %s", url)
}
