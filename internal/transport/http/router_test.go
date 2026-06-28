package http

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRouter_HealthRoutes exercises the health endpoints through the full router
// (middleware chain + routing), not just the handler functions in isolation.
func TestRouter_HealthRoutes(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		pinger     Pinger
		wantStatus int
	}{
		{"liveness", "/healthz", stubPinger{nil}, http.StatusOK},
		{"readiness ready", "/readyz", stubPinger{nil}, http.StatusOK},
		{"readiness down", "/readyz", stubPinger{errors.New("down")}, http.StatusServiceUnavailable},
		{"unknown route", "/nope", stubPinger{nil}, http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(RouterDeps{
				Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
				Pinger: tc.pinger,
			})

			srv := httptest.NewServer(router)
			defer srv.Close()

			resp, err := http.Get(srv.URL + tc.path)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}
