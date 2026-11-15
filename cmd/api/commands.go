package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"kibou/thisisyouryear/internal/auth"
	"kibou/thisisyouryear/internal/config"
	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/platform/mongo"
	"kibou/thisisyouryear/internal/repository"
	"kibou/thisisyouryear/internal/service"
)

const usageText = `Usage:
  api                                   run the server
  api healthcheck                       exit 0 if the local server is ready (used by Docker)
  api admin list-users                  list every account
  api admin reset-password --email X    set a temporary password and sign X out everywhere
  api admin delete-user --email X       permanently delete X's account and all their goals
                        [--yes]         (asks you to type the email first, unless --yes)

In the Pi deployment, run admin commands with:
  docker compose exec api /app/api admin list-users
`

// runCommand executes an operator subcommand and returns the process exit code
// (0 success, 1 failure, 2 usage error).
func runCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	switch args[0] {
	case "healthcheck":
		if err := healthcheck(); err != nil {
			_, _ = fmt.Fprintln(stderr, "unhealthy:", err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, "ok")
		return 0
	case "admin":
		if err := runAdmin(args[1:], stdin, stdout); err != nil {
			_, _ = fmt.Fprintln(stderr, "error:", err)
			if errors.Is(err, errUsage) {
				_, _ = fmt.Fprint(stderr, "\n"+usageText)
				return 2
			}
			return 1
		}
		return 0
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stdout, usageText)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usageText)
		return 2
	}
}

var errUsage = errors.New("invalid usage")

// healthcheck asks the locally running server whether it is ready (it can reach
// MongoDB). The distroless image has no curl, so the Docker healthcheck and the
// Pi's update script call this instead.
func healthcheck() error {
	port := 8080
	if v := os.Getenv("HTTP_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			return fmt.Errorf("invalid HTTP_PORT %q", v)
		}
		port = p
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Always loopback: only the port is configurable, and it is a validated number.
	url := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/readyz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil) //nolint:gosec // G704: fixed loopback host, validated numeric port
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // G704: see above; the request only ever targets 127.0.0.1
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("readyz returned %d", resp.StatusCode)
	}
	return nil
}

// runAdmin connects to the configured database and runs one admin command.
func runAdmin(args []string, in io.Reader, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: missing admin command", errUsage)
	}

	var email string
	var yes bool
	switch args[0] {
	case "list-users":
		if len(args) > 1 {
			return fmt.Errorf("%w: list-users takes no arguments", errUsage)
		}
	case "reset-password":
		fs := flag.NewFlagSet("reset-password", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		fs.StringVar(&email, "email", "", "email of the account to reset")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("%w: %v", errUsage, err)
		}
		if email == "" || fs.NArg() > 0 {
			return fmt.Errorf("%w: reset-password needs exactly --email EMAIL", errUsage)
		}
	case "delete-user":
		fs := flag.NewFlagSet("delete-user", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		fs.StringVar(&email, "email", "", "email of the account to delete")
		fs.BoolVar(&yes, "yes", false, "skip the typed confirmation")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("%w: %v", errUsage, err)
		}
		if email == "" || fs.NArg() > 0 {
			return fmt.Errorf("%w: delete-user needs --email EMAIL (and optionally --yes)", errUsage)
		}
	default:
		return fmt.Errorf("%w: unknown admin command %q", errUsage, args[0])
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	connectCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(connectCtx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoTimeout)
	if err != nil {
		return err
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	admin := service.NewAdminService(service.AdminDeps{
		Users:       repository.NewUserRepository(client, time.Now),
		Sessions:    repository.NewSessionRepository(client, time.Now),
		Topics:      repository.NewTopicRepository(client, time.Now),
		Goals:       repository.NewGoalRepository(client, time.Now),
		Hasher:      auth.NewPasswordHasher(cfg.BcryptCost),
		NewPassword: auth.GenerateTemporaryPassword,
	})

	switch args[0] {
	case "list-users":
		ctx, cancel := opContext()
		defer cancel()
		return listUsers(ctx, admin, out)
	case "reset-password":
		ctx, cancel := opContext()
		defer cancel()
		return resetPassword(ctx, admin, email, out)
	default: // delete-user
		return deleteUser(admin, email, yes, in, out)
	}
}

// opContext bounds one database operation. Each operation gets its own, so time
// spent waiting for the operator to type a confirmation doesn't eat into it.
func opContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func listUsers(ctx context.Context, admin *service.AdminService, out io.Writer) error {
	users, err := admin.ListUsers(ctx)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(tw, "EMAIL\tNAME\tJOINED")
	for _, u := range users {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", u.Email, u.DisplayName, u.CreatedAt.UTC().Format("02-01-2006"))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%d account(s)\n", len(users))
	return err
}

func resetPassword(ctx context.Context, admin *service.AdminService, email string, out io.Writer) error {
	password, err := admin.ResetPassword(ctx, email)
	if domain.IsNotFound(err) {
		return fmt.Errorf("no account with email %q (see: api admin list-users)", email)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Password reset for %s.\nTemporary password: %s\n"+
		"They are now signed out on every device. Send this password privately; it is not shown again.\n",
		domain.NormalizeEmail(email), password)
	return err
}

// accountDeleter is what delete-user needs from AdminService (faked in tests).
type accountDeleter interface {
	Summary(ctx context.Context, email string) (*service.AccountSummary, error)
	DeleteUser(ctx context.Context, email string) (*service.AccountSummary, error)
}

// deleteUser shows what the account holds, asks the operator to type its email
// (unless yes), then deletes it and reports what was removed.
func deleteUser(admin accountDeleter, email string, yes bool, in io.Reader, out io.Writer) error {
	ctx, cancel := opContext()
	sum, err := admin.Summary(ctx, email)
	cancel()
	if domain.IsNotFound(err) {
		return fmt.Errorf("no account with email %q (see: api admin list-users)", email)
	}
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "This permanently deletes %s (%s) and everything in it: %s.\n",
		sum.User.Email, sum.User.DisplayName, describeCounts(sum))

	if !yes {
		_, _ = fmt.Fprint(out, "Type the account's email to confirm: ")
		answer, err := bufio.NewReader(in).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if domain.NormalizeEmail(answer) != sum.User.Email {
			return errors.New("the email you typed doesn't match; nothing was deleted")
		}
	}

	ctx, cancel = opContext()
	defer cancel()
	deleted, err := admin.DeleteUser(ctx, email)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Deleted %s: %s.\n", deleted.User.Email, describeCounts(deleted))
	return err
}

func describeCounts(s *service.AccountSummary) string {
	return fmt.Sprintf("%s, %s, %s",
		plural(s.Topics, "big goal"), plural(s.Goals, "milestone"), plural(s.Sessions, "sign-in session"))
}

func plural(n int64, unit string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}
