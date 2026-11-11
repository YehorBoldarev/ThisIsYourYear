package main

import (
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

In the Pi deployment, run admin commands with:
  docker compose exec api /app/api admin list-users
`

// runCommand executes an operator subcommand and returns the process exit code
// (0 success, 1 failure, 2 usage error).
func runCommand(args []string, stdout, stderr io.Writer) int {
	switch args[0] {
	case "healthcheck":
		if err := healthcheck(); err != nil {
			_, _ = fmt.Fprintln(stderr, "unhealthy:", err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, "ok")
		return 0
	case "admin":
		if err := runAdmin(args[1:], stdout); err != nil {
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
func runAdmin(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: missing admin command", errUsage)
	}

	var email string
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
	default:
		return fmt.Errorf("%w: unknown admin command %q", errUsage, args[0])
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoTimeout)
	if err != nil {
		return err
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	admin := service.NewAdminService(
		repository.NewUserRepository(client, time.Now),
		repository.NewSessionRepository(client, time.Now),
		auth.NewPasswordHasher(cfg.BcryptCost),
		auth.GenerateTemporaryPassword,
	)

	if args[0] == "list-users" {
		return listUsers(ctx, admin, out)
	}
	return resetPassword(ctx, admin, email, out)
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
