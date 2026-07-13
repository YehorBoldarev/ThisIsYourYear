//go:build integration

package http

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A well-formed ObjectID hex that will not exist in a fresh database.
const absentObjectID = "000000000000000000000000"

// TestErrorPaths_ValidationEnvelope covers the 422 surface: field validation,
// malformed JSON, and unknown fields all return the standard envelope.
func TestErrorPaths_ValidationEnvelope(t *testing.T) {
	app := startApp(t)

	t.Run("field validation with per-field detail", func(t *testing.T) {
		resp := app.doJSON(t, http.MethodPost, "/api/v1/auth/register", RegisterRequest{
			Email: "not-an-email", DisplayName: "", Password: "short",
		}, "")
		body := decodeError(t, resp, http.StatusUnprocessableEntity)
		require.Equal(t, "validation_error", body.Error.Code)
		require.NotEmpty(t, body.Error.Fields, "field errors should be reported")
		require.Contains(t, body.Error.Fields, "email")
		require.Contains(t, body.Error.Fields, "password")
	})

	t.Run("malformed JSON", func(t *testing.T) {
		resp := app.doRaw(t, http.MethodPost, "/api/v1/auth/register", []byte("{not json"), "")
		body := decodeError(t, resp, http.StatusUnprocessableEntity)
		require.Equal(t, "validation_error", body.Error.Code)
	})

	t.Run("unknown field", func(t *testing.T) {
		resp := app.doRaw(t, http.MethodPost, "/api/v1/auth/register",
			[]byte(`{"email":"a@b.com","displayName":"A","password":"password123","admin":true}`), "")
		body := decodeError(t, resp, http.StatusUnprocessableEntity)
		require.Equal(t, "validation_error", body.Error.Code)
	})
}

// TestErrorPaths_Unauthorized covers the 401 surface for a protected route.
func TestErrorPaths_Unauthorized(t *testing.T) {
	app := startApp(t)

	cases := map[string]string{
		"no token":      "",
		"garbage token": "not-a-real-jwt",
		"malformed jwt": "aaa.bbb.ccc",
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			resp := app.doJSON(t, http.MethodGet, "/api/v1/me", nil, token)
			body := decodeError(t, resp, http.StatusUnauthorized)
			require.Equal(t, "unauthorized", body.Error.Code)
		})
	}
}

// TestErrorPaths_NotFound covers 404 for both a well-formed-but-absent id and a
// resource owned by another user (ownership is enforced, existence not leaked).
func TestErrorPaths_NotFound(t *testing.T) {
	app := startApp(t)

	t.Run("absent topic", func(t *testing.T) {
		token := app.registerAndLogin(t, "absent@example.com")
		resp := app.doJSON(t, http.MethodGet, "/api/v1/topics/"+absentObjectID, nil, token)
		body := decodeError(t, resp, http.StatusNotFound)
		require.Equal(t, "not_found", body.Error.Code)
	})

	t.Run("another user's topic", func(t *testing.T) {
		alice := app.registerAndLogin(t, "alice-404@example.com")
		bob := app.registerAndLogin(t, "bob-404@example.com")
		topicID := app.createTopic(t, alice, "Alice's private topic")

		// Bob may not read, update, or delete Alice's topic.
		read := app.doJSON(t, http.MethodGet, "/api/v1/topics/"+topicID, nil, bob)
		require.Equal(t, "not_found", decodeError(t, read, http.StatusNotFound).Error.Code)

		del := app.doJSON(t, http.MethodDelete, "/api/v1/topics/"+topicID, nil, bob)
		del.Body.Close()
		require.Equal(t, http.StatusNotFound, del.StatusCode)
	})
}

// TestErrorPaths_Conflict covers 409 for a duplicate email.
func TestErrorPaths_Conflict(t *testing.T) {
	app := startApp(t)
	reg := RegisterRequest{Email: "dupe@example.com", DisplayName: "Dupe", Password: "password123"}

	first := app.doJSON(t, http.MethodPost, "/api/v1/auth/register", reg, "")
	require.Equal(t, http.StatusCreated, first.StatusCode)
	first.Body.Close()

	second := app.doJSON(t, http.MethodPost, "/api/v1/auth/register", reg, "")
	body := decodeError(t, second, http.StatusConflict)
	require.Equal(t, "conflict", body.Error.Code)
}

// TestErrorPaths_RateLimited covers 429 when the auth limit is exceeded.
func TestErrorPaths_RateLimited(t *testing.T) {
	// One request per (long) window makes the second call trip the limit.
	app := startAppWithRateLimit(t, RateLimitConfig{AuthRequests: 1, AuthWindow: time.Minute})

	login := LoginRequest{Email: "nobody@example.com", Password: "whatever123"}

	// First auth request is allowed through (and fails auth on its own merits).
	first := app.doJSON(t, http.MethodPost, "/api/v1/auth/login", login, "")
	require.NotEqual(t, http.StatusTooManyRequests, first.StatusCode)
	first.Body.Close()

	// Second within the window is throttled with the standard envelope.
	second := app.doJSON(t, http.MethodPost, "/api/v1/auth/login", login, "")
	require.NotEmpty(t, second.Header.Get("Retry-After"))
	body := decodeError(t, second, http.StatusTooManyRequests)
	require.Equal(t, "rate_limited", body.Error.Code)
}

// TestErrorPaths_MethodNotAllowed covers 405 on a known path with a wrong method.
func TestErrorPaths_MethodNotAllowed(t *testing.T) {
	app := startApp(t)
	resp := app.doJSON(t, http.MethodPost, "/healthz", nil, "")
	resp.Body.Close()
	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// decodeError asserts the status code and decodes the standard error envelope.
func decodeError(t *testing.T, resp *http.Response, wantStatus int) errorBody {
	t.Helper()
	require.Equal(t, wantStatus, resp.StatusCode)
	var body errorBody
	decodeBody(t, resp, &body)
	return body
}

// doRaw sends a request with a raw (already-encoded) body, so tests can exercise
// malformed or unexpected payloads that doJSON's marshaling would never produce.
func (a *testApp) doRaw(t *testing.T, method, path string, body []byte, bearer string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, a.server.URL+path, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := a.client.Do(req)
	require.NoError(t, err)
	return resp
}
