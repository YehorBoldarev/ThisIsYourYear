//go:build integration

package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthFlow_HappyPath(t *testing.T) {
	app := startApp(t)

	// Register.
	resp := app.doJSON(t, http.MethodPost, "/api/v1/auth/register", RegisterRequest{
		Email:       "alice@example.com",
		DisplayName: "Alice",
		Password:    "password123",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created UserResponse
	decodeBody(t, resp, &created)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "alice@example.com", created.Email)

	// Login (remember me) sets the refresh cookie and returns an access token.
	resp = app.doJSON(t, http.MethodPost, "/api/v1/auth/login", LoginRequest{
		Email: "alice@example.com", Password: "password123", RememberMe: true,
	}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, refreshCookieValue(resp), "login sets a refresh cookie")
	var login TokenResponse
	decodeBody(t, resp, &login)
	require.NotEmpty(t, login.AccessToken)

	// /me with the access token returns the profile.
	resp = app.doJSON(t, http.MethodGet, "/api/v1/me", nil, login.AccessToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var me UserResponse
	decodeBody(t, resp, &me)
	require.Equal(t, "alice@example.com", me.Email)
	resp.Body.Close()

	// /me without a token is unauthorized.
	resp = app.doJSON(t, http.MethodGet, "/api/v1/me", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()

	// Refresh (cookie auto-sent by the jar) rotates the token.
	resp = app.doJSON(t, http.MethodPost, "/api/v1/auth/refresh", nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var refreshed TokenResponse
	decodeBody(t, resp, &refreshed)
	require.NotEmpty(t, refreshed.AccessToken)

	// The rotated access token works against a protected route.
	resp = app.doJSON(t, http.MethodGet, "/api/v1/me", nil, refreshed.AccessToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Logout clears the cookie and revokes the session.
	resp = app.doJSON(t, http.MethodPost, "/api/v1/auth/logout", nil, "")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// With the cookie cleared, refresh is unauthorized.
	resp = app.doJSON(t, http.MethodPost, "/api/v1/auth/refresh", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestAuthFlow_RefreshReuseRevokesLineage(t *testing.T) {
	app := startApp(t)

	resp := app.doJSON(t, http.MethodPost, "/api/v1/auth/register", RegisterRequest{
		Email: "bob@example.com", DisplayName: "Bob", Password: "password123",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp = app.doJSON(t, http.MethodPost, "/api/v1/auth/login", LoginRequest{
		Email: "bob@example.com", Password: "password123", RememberMe: true,
	}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	first := refreshCookieValue(resp)
	require.NotEmpty(t, first)
	resp.Body.Close()

	// Rotate: the jar sends `first`, the server returns `second`.
	resp = app.doJSON(t, http.MethodPost, "/api/v1/auth/refresh", nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	second := refreshCookieValue(resp)
	require.NotEmpty(t, second)
	require.NotEqual(t, first, second, "the refresh token is rotated")
	resp.Body.Close()

	// Replaying the rotated-away `first` token is detected as reuse.
	resp = app.rawRefresh(t, first)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()

	// Reuse revoked the whole lineage, so even `second` no longer works.
	resp = app.rawRefresh(t, second)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestAuthFlow_ErrorCases(t *testing.T) {
	app := startApp(t)

	// Invalid email -> 422 with field detail.
	resp := app.doJSON(t, http.MethodPost, "/api/v1/auth/register", RegisterRequest{
		Email: "not-an-email", DisplayName: "X", Password: "password123",
	}, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	resp.Body.Close()

	// Create a real account.
	resp = app.doJSON(t, http.MethodPost, "/api/v1/auth/register", RegisterRequest{
		Email: "carol@example.com", DisplayName: "Carol", Password: "password123",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Duplicate email -> 409.
	resp = app.doJSON(t, http.MethodPost, "/api/v1/auth/register", RegisterRequest{
		Email: "carol@example.com", DisplayName: "Carol2", Password: "password456",
	}, "")
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()

	// Wrong password -> 401.
	resp = app.doJSON(t, http.MethodPost, "/api/v1/auth/login", LoginRequest{
		Email: "carol@example.com", Password: "wrong",
	}, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}
