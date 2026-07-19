package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func spaTestFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":              {Data: []byte("<!doctype html><title>app shell</title>")},
		"favicon.svg":             {Data: []byte("<svg/>")},
		"assets/index-abc123.js":  {Data: []byte("console.log(1)")},
		"assets/index-abc123.css": {Data: []byte(".x{}")},
	}
}

func newTestSPAHandler() http.HandlerFunc {
	return newSPAHandler(spaTestFS(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func doSPA(t *testing.T, method, target string) *http.Response {
	t.Helper()
	rec := httptest.NewRecorder()
	newTestSPAHandler()(rec, httptest.NewRequest(method, target, nil))
	return rec.Result()
}

func TestSPAHandler_ServesHashedAssetWithImmutableCache(t *testing.T) {
	resp := doSPA(t, http.MethodGet, "/assets/index-abc123.js")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Cache-Control"), "immutable")

	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, "console.log(1)", string(body))
}

func TestSPAHandler_DeepLinkFallsBackToIndex(t *testing.T) {
	resp := doSPA(t, http.MethodGet, "/achievements")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "text/html")
	require.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))

	body, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(body), "app shell")
}

func TestSPAHandler_RootServesIndex(t *testing.T) {
	resp := doSPA(t, http.MethodGet, "/")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(body), "app shell")
}

func TestSPAHandler_UnmatchedAPIReturnsJSON404(t *testing.T) {
	resp := doSPA(t, http.MethodGet, "/api/v1/nope")
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var body errorBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "not_found", body.Error.Code)
}

func TestSPAHandler_SetsSecurityHeadersOnHTML(t *testing.T) {
	resp := doSPA(t, http.MethodGet, "/")
	defer resp.Body.Close()

	require.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	require.Equal(t, "same-origin", resp.Header.Get("Referrer-Policy"))

	csp := resp.Header.Get("Content-Security-Policy")
	require.Contains(t, csp, "default-src 'self'")
	require.Contains(t, csp, "script-src 'self'")
	require.NotContains(t, csp, "script-src 'self' 'unsafe-inline'") // scripts stay strict
}

func TestSPAHandler_NonGETIsNotFound(t *testing.T) {
	resp := doSPA(t, http.MethodPost, "/achievements")
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
