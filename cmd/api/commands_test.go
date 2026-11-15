package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/service"
)

func TestRunCommand_UsageErrors(t *testing.T) {
	cases := map[string][]string{
		"unknown command":          {"bogus"},
		"admin without subcommand": {"admin"},
		"unknown admin command":    {"admin", "drop-database"},
		"reset without email":      {"admin", "reset-password"},
		"reset with stray arg":     {"admin", "reset-password", "--email", "a@b.co", "extra"},
		"list-users with args":     {"admin", "list-users", "now"},
		"delete without email":     {"admin", "delete-user"},
		"delete with stray arg":    {"admin", "delete-user", "--email", "a@b.co", "now"},
		"delete with bad flag":     {"admin", "delete-user", "--email", "a@b.co", "--force"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			require.Equal(t, 2, runCommand(args, strings.NewReader(""), &stdout, &stderr))
			require.Contains(t, stderr.String(), "Usage:")
		})
	}
}

func TestRunCommand_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, runCommand([]string{"help"}, strings.NewReader(""), &stdout, &stderr))
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
	require.Equal(t, 0, runCommand([]string{"healthcheck"}, strings.NewReader(""), &stdout, &stderr))
	require.Equal(t, "ok\n", stdout.String())

	status.Store(http.StatusServiceUnavailable)
	require.Equal(t, 1, runCommand([]string{"healthcheck"}, strings.NewReader(""), &stdout, &stderr))
	require.Contains(t, stderr.String(), "readyz returned 503")
}

// fakeDeleter stands in for AdminService in the delete-user flow.
type fakeDeleter struct {
	summary *service.AccountSummary
	deleted bool
}

func (f *fakeDeleter) Summary(_ context.Context, email string) (*service.AccountSummary, error) {
	if domain.NormalizeEmail(email) != f.summary.User.Email {
		return nil, domain.ErrNotFound
	}
	return f.summary, nil
}

func (f *fakeDeleter) DeleteUser(_ context.Context, _ string) (*service.AccountSummary, error) {
	f.deleted = true
	return f.summary, nil
}

func newFakeDeleter() *fakeDeleter {
	return &fakeDeleter{summary: &service.AccountSummary{
		User:   &domain.User{Email: "fam@example.com", DisplayName: "Fam"},
		Topics: 3, Goals: 1, Sessions: 2,
	}}
}

func TestDeleteUser_TypedConfirmation(t *testing.T) {
	cases := map[string]struct {
		input      string
		wantDelete bool
	}{
		"exact email":                    {"fam@example.com\n", true},
		"case and spaces don't matter":   {"  FAM@Example.com \n", true},
		"no trailing newline (piped)":    {"fam@example.com", true},
		"different email":                {"other@example.com\n", false},
		"empty answer":                   {"\n", false},
		"no input at all (closed stdin)": {"", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fake := newFakeDeleter()
			var out bytes.Buffer
			err := deleteUser(fake, "fam@example.com", false, strings.NewReader(tc.input), &out)
			require.Equal(t, tc.wantDelete, fake.deleted)
			require.Contains(t, out.String(), "3 big goals, 1 milestone, 2 sign-in sessions")
			require.Contains(t, out.String(), "Type the account's email to confirm")
			if tc.wantDelete {
				require.NoError(t, err)
				require.Contains(t, out.String(), "Deleted fam@example.com")
			} else {
				require.ErrorContains(t, err, "nothing was deleted")
			}
		})
	}
}

func TestDeleteUser_YesSkipsTheQuestion(t *testing.T) {
	fake := newFakeDeleter()
	var out bytes.Buffer
	require.NoError(t, deleteUser(fake, "fam@example.com", true, strings.NewReader(""), &out))
	require.True(t, fake.deleted)
	require.NotContains(t, out.String(), "Type the account's email")
}

func TestDeleteUser_UnknownAccount(t *testing.T) {
	fake := newFakeDeleter()
	err := deleteUser(fake, "nobody@example.com", true, strings.NewReader(""), &bytes.Buffer{})
	require.ErrorContains(t, err, `no account with email "nobody@example.com"`)
	require.False(t, fake.deleted)
}
