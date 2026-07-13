//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"

	"kibou/thisisyouryear/internal/auth"
	"kibou/thisisyouryear/internal/platform/mongo"
	"kibou/thisisyouryear/internal/repository"
	"kibou/thisisyouryear/internal/service"
)

const refreshCookieName = "refresh_token"

// testApp is the whole service wired against a real MongoDB container and served
// over an httptest server, with a cookie-jar client so the refresh cookie flows
// exactly as a browser would carry it.
type testApp struct {
	server *httptest.Server
	client *http.Client
}

// startApp starts the full service with auth rate limiting disabled, which is
// what every flow test wants (they register/login many users in quick
// succession). Tests that need throttling use startAppWithRateLimit.
func startApp(t *testing.T) *testApp {
	return startAppWithRateLimit(t, RateLimitConfig{})
}

// startAppWithRateLimit starts the full service with the given auth rate-limit
// configuration (a zero value disables limiting).
func startAppWithRateLimit(t *testing.T, rl RateLimitConfig) *testApp {
	t.Helper()
	ctx := context.Background()

	container, err := mongodb.Run(ctx, "mongo:7")
	require.NoError(t, err, "start mongo container")
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	uri, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(ctx, uri, "thisisyouryear_test", 30*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	// All repositories register their indexes on construction, so build them
	// before EnsureIndexes runs.
	userRepo := repository.NewUserRepository(client, time.Now)
	sessionRepo := repository.NewSessionRepository(client, time.Now)
	topicRepo := repository.NewTopicRepository(client, time.Now)
	goalRepo := repository.NewGoalRepository(client, time.Now)
	require.NoError(t, client.EnsureIndexes(ctx))

	tokens := auth.NewTokenIssuer("integration-secret", 15*time.Minute)
	authService := service.NewAuthService(
		userRepo, sessionRepo,
		auth.NewPasswordHasher(10), // low cost keeps tests fast
		tokens,
		service.AuthConfig{RefreshTTLRemember: 744 * time.Hour, RefreshTTLDefault: 24 * time.Hour},
	)

	topicService := service.NewTopicService(topicRepo, goalRepo)
	router := NewRouter(RouterDeps{
		Logger:           quietLogger(),
		Pinger:           client,
		AuthService:      authService,
		UserService:      service.NewUserService(userRepo),
		TopicService:     topicService,
		GoalService:      service.NewGoalService(goalRepo, topicRepo),
		DashboardService: service.NewDashboardService(topicService, time.Now),
		TokenVerifier:    tokens,
		RateLimit:        rl,
		// Secure=false so the cookie jar sends the refresh cookie over the
		// httptest http:// server (a Secure cookie is withheld on plain HTTP).
		RefreshCookie: RefreshCookieConfig{Name: refreshCookieName, Path: "/api/v1/auth", Secure: false},
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	return &testApp{server: srv, client: &http.Client{Jar: jar}}
}

// doJSON performs a request through the jar-backed client, marshaling body (if
// any) as JSON and attaching a bearer token (if any).
func (a *testApp) doJSON(t *testing.T, method, path string, body any, bearer string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, a.server.URL+path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := a.client.Do(req)
	require.NoError(t, err)
	return resp
}

// rawRefresh posts to /refresh with an explicit refresh cookie, bypassing the
// jar, so a specific (possibly rotated-away) token can be replayed.
func (a *testApp) rawRefresh(t *testing.T, cookieValue string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, a.server.URL+"/api/v1/auth/refresh", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: cookieValue})
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func refreshCookieValue(resp *http.Response) string {
	for _, c := range resp.Cookies() {
		if c.Name == refreshCookieName {
			return c.Value
		}
	}
	return ""
}

func decodeBody(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(dst))
}

// registerAndLogin creates an account and returns a valid bearer access token for
// it. Each call uses its own client so different users don't share a cookie jar.
func (a *testApp) registerAndLogin(t *testing.T, email string) string {
	t.Helper()
	resp := a.doJSON(t, http.MethodPost, "/api/v1/auth/register", RegisterRequest{
		Email: email, DisplayName: "User", Password: "password123",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp = a.doJSON(t, http.MethodPost, "/api/v1/auth/login", LoginRequest{
		Email: email, Password: "password123",
	}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var login TokenResponse
	decodeBody(t, resp, &login)
	require.NotEmpty(t, login.AccessToken)
	return login.AccessToken
}

// settingDate builds a request date field that sets the given DD-MM-YYYY value.
func settingDate(s string) *optionalDate {
	t, _ := time.Parse(dateLayout, s)
	t = t.UTC()
	return &optionalDate{value: &t}
}

// clearingDate builds a request date field that clears the value (sends "").
func clearingDate() *optionalDate {
	return &optionalDate{}
}

// createTopic creates a topic and returns its id.
func (a *testApp) createTopic(t *testing.T, token, title string) string {
	t.Helper()
	resp := a.doJSON(t, http.MethodPost, "/api/v1/topics", CreateTopicRequest{Title: title}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var topic TopicResponse
	decodeBody(t, resp, &topic)
	require.NotEmpty(t, topic.ID)
	return topic.ID
}
