package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/auth"
	"kibou/thisisyouryear/internal/domain"
)

type stubVerifier struct {
	userID string
	err    error
}

func (s stubVerifier) Verify(string) (auth.Claims, error) {
	if s.err != nil {
		return auth.Claims{}, s.err
	}
	return auth.Claims{UserID: s.userID}, nil
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestAuthenticator_MissingHeader(t *testing.T) {
	a := NewAuthenticator(stubVerifier{userID: "u1"}, quietLogger())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)

	a.Middleware(failIfCalled(t)).ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthenticator_InvalidToken(t *testing.T) {
	a := NewAuthenticator(stubVerifier{err: domain.ErrUnauthenticated}, quietLogger())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer bad-token")

	a.Middleware(failIfCalled(t)).ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthenticator_ValidTokenInjectsUserID(t *testing.T) {
	a := NewAuthenticator(stubVerifier{userID: "user-42"}, quietLogger())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer good-token")

	var gotID string
	var gotOK bool
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotID, gotOK = UserID(r.Context())
	})
	a.Middleware(next).ServeHTTP(rec, req)

	require.True(t, gotOK)
	require.Equal(t, "user-42", gotID)
}

func TestBearerToken(t *testing.T) {
	cases := map[string]struct {
		header string
		want   string
		wantOK bool
	}{
		"valid":            {"Bearer abc", "abc", true},
		"case-insensitive": {"bearer abc", "abc", true},
		"empty token":      {"Bearer ", "", false},
		"missing prefix":   {"abc", "", false},
		"empty header":     {"", "", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			got, ok := bearerToken(req)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

// failIfCalled returns a handler that fails the test if the request reaches it,
// used to assert the middleware short-circuits unauthorized requests.
func failIfCalled(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("next handler must not be called for an unauthorized request")
	})
}
