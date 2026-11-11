//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/auth"
)

// Self-hosting (plan C3): invite-only registration and the client IP recorded
// behind a trusted reverse proxy, exercised over real HTTP against MongoDB.

func TestRegistrationInfo_OpenByDefault(t *testing.T) {
	app := startApp(t)

	resp := app.doJSON(t, http.MethodGet, "/api/v1/auth/registration", nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var info RegistrationInfoResponse
	decodeBody(t, resp, &info)
	require.False(t, info.InviteRequired)

	app.registerAndLogin(t, "open@example.com") // no invite code needed
}

func TestInviteOnlyRegistration(t *testing.T) {
	app := startAppWith(t, appOptions{InviteCode: "sunflower-2027"})

	resp := app.doJSON(t, http.MethodGet, "/api/v1/auth/registration", nil, "")
	var info RegistrationInfoResponse
	decodeBody(t, resp, &info)
	require.True(t, info.InviteRequired)

	register := func(code string) *http.Response {
		return app.doJSON(t, http.MethodPost, "/api/v1/auth/register", RegisterRequest{
			Email: "fam@example.com", DisplayName: "Fam", Password: "password123", InviteCode: code,
		}, "")
	}

	body := decodeError(t, register(""), http.StatusUnprocessableEntity)
	require.Equal(t, "validation_error", body.Error.Code)
	require.Equal(t, "is required", body.Error.Fields["inviteCode"])

	body = decodeError(t, register("sunflower-2026"), http.StatusUnprocessableEntity)
	require.Equal(t, "is not valid", body.Error.Fields["inviteCode"])

	resp = register("sunflower-2027")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()
}

// loginWithForwardedFor logs in with an X-Forwarded-For header and returns the
// IP stored on the resulting refresh session.
func loginWithForwardedFor(t *testing.T, app *testApp, email, forwarded string) string {
	t.Helper()
	payload, err := json.Marshal(LoginRequest{Email: email, Password: "password123"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, app.server.URL+"/api/v1/auth/login", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", forwarded)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	raw := refreshCookieValue(resp)
	require.NotEmpty(t, raw)
	session, err := app.sessions.ByTokenHash(context.Background(), auth.HashRefreshToken(raw))
	require.NoError(t, err)
	return session.IP
}

func TestClientIP_TrustedProxyHeaderLandsOnSession(t *testing.T) {
	// The httptest server is reached over loopback, so loopback plays the role
	// of the Docker network gateway that tailscaled's requests arrive from.
	app := startAppWith(t, appOptions{ClientIP: ClientIPConfig{
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32"), netip.MustParsePrefix("::1/128")},
		Header:         "X-Forwarded-For",
	}})
	app.registerAndLogin(t, "proxied@example.com")

	require.Equal(t, "198.51.100.7", loginWithForwardedFor(t, app, "proxied@example.com", "198.51.100.7"))
}

func TestClientIP_UntrustedHeaderIgnored(t *testing.T) {
	app := startApp(t) // no trusted proxies
	app.registerAndLogin(t, "direct@example.com")

	ip := loginWithForwardedFor(t, app, "direct@example.com", "198.51.100.7")
	require.NotEqual(t, "198.51.100.7", ip, "a spoofed header must not reach the session")
	require.True(t, netip.MustParseAddr(ip).IsLoopback(), "the session records the real peer (%s)", ip)
}
