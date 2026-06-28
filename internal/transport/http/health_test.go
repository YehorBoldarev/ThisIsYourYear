package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// stubPinger is a test double for the Pinger dependency.
type stubPinger struct{ err error }

func (s stubPinger) Ping(context.Context) error { return s.err }

func TestHealthHandler_AlwaysOK(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	healthHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", decodeStatus(t, rec))
}

func TestReadinessHandler_Ready(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	readinessHandler(stubPinger{err: nil})(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ready", decodeStatus(t, rec))
}

func TestReadinessHandler_Unavailable(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	readinessHandler(stubPinger{err: errors.New("connection refused")})(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, "unavailable", decodeStatus(t, rec))
}

func decodeStatus(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	return body["status"]
}
