// Package config loads and validates application configuration from the
// environment, following 12-factor conventions. No configuration values are
// baked into the binary or images; everything comes from environment variables
// (optionally seeded from a local .env file during development).
package config

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Environment enumerates the supported runtime environments.
type Environment string

const (
	// EnvDevelopment relaxes defaults for local work (loads .env, allows the dev JWT secret).
	EnvDevelopment Environment = "development"
	// EnvProduction enforces production-only checks (e.g. a real JWT_SECRET) and ignores .env.
	EnvProduction Environment = "production"
)

// minInviteCodeLen is the shortest accepted REGISTRATION_INVITE_CODE.
const minInviteCodeLen = 8

// devJWTSecret is the throwaway signing secret used when none is configured. It
// exists only so the service runs out-of-the-box in development; using it in
// production is rejected by validation. It is not a real credential.
//
//nolint:gosec // G101: intentional non-secret development placeholder, rejected in production by validate().
const devJWTSecret = "dev-insecure-jwt-secret-change-me"

// Config holds all runtime configuration for the API service.
//
// Defaults are chosen so that a developer can run the service locally with zero
// configuration against a Mongo instance started via docker-compose.
type Config struct {
	AppEnv Environment `env:"APP_ENV" envDefault:"development"`

	HTTPPort         int           `env:"HTTP_PORT" envDefault:"8080"`
	HTTPReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"10s"`
	HTTPWriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"15s"`
	HTTPIdleTimeout  time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s"`
	ShutdownTimeout  time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`

	MongoURI      string        `env:"MONGO_URI" envDefault:"mongodb://localhost:27017"`
	MongoDatabase string        `env:"MONGO_DATABASE" envDefault:"thisisyouryear"`
	MongoTimeout  time.Duration `env:"MONGO_TIMEOUT" envDefault:"10s"`

	// Authentication.
	JWTSecret          string        `env:"JWT_SECRET" envDefault:"dev-insecure-jwt-secret-change-me"`
	AccessTokenTTL     time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTTLRemember time.Duration `env:"REFRESH_TTL_REMEMBER" envDefault:"744h"` // 31 days
	RefreshTTLDefault  time.Duration `env:"REFRESH_TTL_DEFAULT" envDefault:"24h"`
	BcryptCost         int           `env:"BCRYPT_COST" envDefault:"12"`

	// Refresh-cookie and CORS settings for the browser SPA.
	CookieDomain       string   `env:"COOKIE_DOMAIN" envDefault:""`
	CookieSecure       bool     `env:"COOKIE_SECURE" envDefault:"true"`
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envSeparator:","`

	// Rate limiting for the public auth endpoints, applied per client IP. A
	// request count of 0 disables limiting entirely (used by tests and by anyone
	// who wants to opt out). The defaults are generous for a human but stop
	// automated brute-force against register/login/refresh.
	RateLimitAuthRequests int           `env:"RATE_LIMIT_AUTH_REQUESTS" envDefault:"20"`
	RateLimitAuthWindow   time.Duration `env:"RATE_LIMIT_AUTH_WINDOW" envDefault:"1m"`

	// Client IP behind a reverse proxy. Requests arriving from an address in
	// TrustedProxies (IPs or CIDRs) take the client IP from ClientIPHeader, which
	// must hold exactly one IP (the proxy overwrites it; chains are rejected).
	// Empty TrustedProxies means forwarding headers are never trusted and the
	// client IP is the direct peer address.
	TrustedProxies []string `env:"TRUSTED_PROXIES" envSeparator:","`
	ClientIPHeader string   `env:"CLIENT_IP_HEADER" envDefault:"X-Forwarded-For"`

	// RegistrationInviteCode, when set, makes sign-up invite-only: registration
	// must present this code. Empty means open registration.
	RegistrationInviteCode string `env:"REGISTRATION_INVITE_CODE"`

	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

// IsProduction reports whether the service is running in the production environment.
func (c Config) IsProduction() bool {
	return c.AppEnv == EnvProduction
}

// Load reads configuration from the environment and validates it.
//
// In non-production environments it first attempts to load a local .env file so
// developers can keep their settings out of their shell profile; a missing .env
// file is not an error. In production, configuration must come from the real
// environment and .env files are ignored.
func Load() (Config, error) {
	var cfg Config

	// Peek at APP_ENV before deciding whether to consult a .env file, so that a
	// stray .env can never influence a production deployment.
	if probe := loadEnvForProbe(); probe != EnvProduction {
		// Best-effort: absence of a .env file is expected and not an error.
		_ = godotenv.Load()
	}

	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}

	// A comma-separated env var can yield empty entries (e.g. an unset or
	// trailing-comma value); drop them so validation and CORS see a clean list.
	cfg.CORSAllowedOrigins = cleanOrigins(cfg.CORSAllowedOrigins)
	cfg.TrustedProxies = cleanOrigins(cfg.TrustedProxies)
	cfg.ClientIPHeader = strings.TrimSpace(cfg.ClientIPHeader)
	cfg.RegistrationInviteCode = strings.TrimSpace(cfg.RegistrationInviteCode)

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// TrustedProxyPrefixes returns TRUSTED_PROXIES as network prefixes (a bare IP
// becomes a single-address prefix). Load has already validated every entry, so
// unparsable entries cannot occur on a loaded Config and are skipped.
func (c Config) TrustedProxyPrefixes() []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(c.TrustedProxies))
	for _, entry := range c.TrustedProxies {
		if p, err := parsePrefix(entry); err == nil {
			prefixes = append(prefixes, p)
		}
	}
	return prefixes
}

// parsePrefix accepts either a CIDR ("172.30.42.0/24") or a bare IP
// ("172.30.42.1"), returning the equivalent prefix.
func parsePrefix(entry string) (netip.Prefix, error) {
	if strings.Contains(entry, "/") {
		p, err := netip.ParsePrefix(entry)
		if err != nil {
			return netip.Prefix{}, err
		}
		return p.Masked(), nil
	}
	addr, err := netip.ParseAddr(entry)
	if err != nil {
		return netip.Prefix{}, err
	}
	addr = addr.Unmap()
	return netip.PrefixFrom(addr, addr.BitLen()), nil
}

// cleanOrigins trims whitespace and removes empty entries from a list of origins.
func cleanOrigins(origins []string) []string {
	cleaned := make([]string, 0, len(origins))
	for _, o := range origins {
		if o = strings.TrimSpace(o); o != "" {
			cleaned = append(cleaned, o)
		}
	}
	return cleaned
}

func (c Config) validate() error {
	switch c.AppEnv {
	case EnvDevelopment, EnvProduction:
	default:
		return fmt.Errorf("invalid APP_ENV %q: must be %q or %q", c.AppEnv, EnvDevelopment, EnvProduction)
	}

	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP_PORT %d: must be in range 1-65535", c.HTTPPort)
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid LOG_LEVEL %q: must be one of debug, info, warn, error", c.LogLevel)
	}

	if c.MongoURI == "" {
		return fmt.Errorf("MONGO_URI must not be empty")
	}
	if c.MongoDatabase == "" {
		return fmt.Errorf("MONGO_DATABASE must not be empty")
	}

	if err := c.validateAuth(); err != nil {
		return err
	}

	for _, entry := range c.TrustedProxies {
		if _, err := parsePrefix(entry); err != nil {
			return fmt.Errorf("invalid TRUSTED_PROXIES entry %q: must be an IP or CIDR", entry)
		}
	}
	if len(c.TrustedProxies) > 0 && c.ClientIPHeader == "" {
		return fmt.Errorf("CLIENT_IP_HEADER must be set when TRUSTED_PROXIES is set")
	}

	// Rate limiting: a negative count is nonsensical; 0 is the explicit "off"
	// switch. A positive count needs a positive window to divide requests into.
	if c.RateLimitAuthRequests < 0 {
		return fmt.Errorf("invalid RATE_LIMIT_AUTH_REQUESTS %d: must be >= 0 (0 disables limiting)", c.RateLimitAuthRequests)
	}
	if c.RateLimitAuthRequests > 0 && c.RateLimitAuthWindow <= 0 {
		return fmt.Errorf("RATE_LIMIT_AUTH_WINDOW must be positive when rate limiting is enabled")
	}

	return nil
}

// validateAuth checks the authentication- and browser-facing settings.
func (c Config) validateAuth() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET must not be empty")
	}
	// The insecure development default must never be used to sign real tokens.
	if c.IsProduction() && c.JWTSecret == devJWTSecret {
		return fmt.Errorf("JWT_SECRET must be set to a strong secret in production")
	}

	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be positive")
	}
	if c.RefreshTTLRemember <= 0 {
		return fmt.Errorf("REFRESH_TTL_REMEMBER must be positive")
	}
	if c.RefreshTTLDefault <= 0 {
		return fmt.Errorf("REFRESH_TTL_DEFAULT must be positive")
	}

	// bcrypt accepts costs in [4, 31]; below 10 is too weak for passwords.
	if c.BcryptCost < 10 || c.BcryptCost > 31 {
		return fmt.Errorf("invalid BCRYPT_COST %d: must be in range 10-31", c.BcryptCost)
	}

	// An invite code is the only gate on sign-up; a short one is guessable.
	if c.RegistrationInviteCode != "" && len(c.RegistrationInviteCode) < minInviteCodeLen {
		return fmt.Errorf("REGISTRATION_INVITE_CODE must be at least %d characters (or empty for open registration)", minInviteCodeLen)
	}

	// The refresh cookie is sent with credentials, so a wildcard CORS origin is
	// both insecure and rejected by browsers; require explicit origins.
	for _, o := range c.CORSAllowedOrigins {
		if o == "*" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS must list explicit origins, not %q (credentialed requests forbid a wildcard)", o)
		}
	}

	return nil
}

// loadEnvForProbe returns the APP_ENV value already present in the process
// environment (before any .env file is loaded), defaulting to development.
func loadEnvForProbe() Environment {
	var probe struct {
		AppEnv Environment `env:"APP_ENV" envDefault:"development"`
	}
	if err := env.Parse(&probe); err != nil {
		return EnvDevelopment
	}
	return probe.AppEnv
}
