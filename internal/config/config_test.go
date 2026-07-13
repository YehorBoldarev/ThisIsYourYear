package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure a clean environment so only defaults apply.
	clearKnownEnv(t)

	cfg, err := Load()
	require.NoError(t, err)

	require.Equal(t, EnvDevelopment, cfg.AppEnv)
	require.Equal(t, 8080, cfg.HTTPPort)
	require.Equal(t, "mongodb://localhost:27017", cfg.MongoURI)
	require.Equal(t, "thisisyouryear", cfg.MongoDatabase)
	require.Equal(t, "info", cfg.LogLevel)
	require.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
	require.False(t, cfg.IsProduction())

	// Auth defaults.
	require.Equal(t, 15*time.Minute, cfg.AccessTokenTTL)
	require.Equal(t, 744*time.Hour, cfg.RefreshTTLRemember)
	require.Equal(t, 24*time.Hour, cfg.RefreshTTLDefault)
	require.Equal(t, 12, cfg.BcryptCost)
	require.True(t, cfg.CookieSecure)
	require.Empty(t, cfg.CORSAllowedOrigins)

	// Rate-limit defaults.
	require.Equal(t, 20, cfg.RateLimitAuthRequests)
	require.Equal(t, time.Minute, cfg.RateLimitAuthWindow)
}

func TestLoad_Overrides(t *testing.T) {
	clearKnownEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("MONGO_DATABASE", "custom")
	t.Setenv("SHUTDOWN_TIMEOUT", "30s")
	// Production requires a real signing secret.
	t.Setenv("JWT_SECRET", "a-strong-production-secret")

	cfg, err := Load()
	require.NoError(t, err)

	require.Equal(t, EnvProduction, cfg.AppEnv)
	require.Equal(t, 9090, cfg.HTTPPort)
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, "custom", cfg.MongoDatabase)
	require.Equal(t, 30*time.Second, cfg.ShutdownTimeout)
	require.True(t, cfg.IsProduction())
}

func TestLoad_InvalidValues(t *testing.T) {
	// Only non-empty overrides are exercised here: caarlos0/env treats an empty
	// environment variable as unset and falls back to the default, so empty
	// values cannot reach validation via the environment.
	tests := map[string]map[string]string{
		"invalid app env":      {"APP_ENV": "staging"},
		"port too low":         {"HTTP_PORT": "0"},
		"port too high":        {"HTTP_PORT": "70000"},
		"invalid log level":    {"LOG_LEVEL": "verbose"},
		"bcrypt cost too low":  {"BCRYPT_COST": "4"},
		"bcrypt cost too high": {"BCRYPT_COST": "40"},
		"prod default secret":  {"APP_ENV": "production", "JWT_SECRET": "dev-insecure-jwt-secret-change-me"},
		"wildcard cors origin": {"CORS_ALLOWED_ORIGINS": "*"},
		"negative rate limit":  {"RATE_LIMIT_AUTH_REQUESTS": "-1"},
	}

	for name, overrides := range tests {
		t.Run(name, func(t *testing.T) {
			clearKnownEnv(t)
			for k, v := range overrides {
				t.Setenv(k, v)
			}
			_, err := Load()
			require.Error(t, err)
		})
	}
}

// TestValidate covers the validation rules directly, including the empty-string
// guards that are unreachable through environment parsing but remain as defense.
func TestValidate(t *testing.T) {
	base := func() Config {
		return Config{
			AppEnv:             EnvDevelopment,
			HTTPPort:           8080,
			LogLevel:           "info",
			MongoURI:           "mongodb://localhost:27017",
			MongoDatabase:      "thisisyouryear",
			JWTSecret:          "some-secret",
			AccessTokenTTL:     15 * time.Minute,
			RefreshTTLRemember: 744 * time.Hour,
			RefreshTTLDefault:  24 * time.Hour,
			BcryptCost:         12,
		}
	}

	require.NoError(t, base().validate())

	emptyURI := base()
	emptyURI.MongoURI = ""
	require.Error(t, emptyURI.validate())

	emptyDB := base()
	emptyDB.MongoDatabase = ""
	require.Error(t, emptyDB.validate())

	emptySecret := base()
	emptySecret.JWTSecret = ""
	require.Error(t, emptySecret.validate())

	nonPositiveTTL := base()
	nonPositiveTTL.AccessTokenTTL = 0
	require.Error(t, nonPositiveTTL.validate())

	prodDefaultSecret := base()
	prodDefaultSecret.AppEnv = EnvProduction
	prodDefaultSecret.JWTSecret = devJWTSecret
	require.Error(t, prodDefaultSecret.validate())

	// Rate limiting: disabled (0 requests) is valid regardless of window.
	disabledLimit := base()
	disabledLimit.RateLimitAuthRequests = 0
	disabledLimit.RateLimitAuthWindow = 0
	require.NoError(t, disabledLimit.validate())

	// Enabled but missing a window is invalid.
	enabledNoWindow := base()
	enabledNoWindow.RateLimitAuthRequests = 5
	enabledNoWindow.RateLimitAuthWindow = 0
	require.Error(t, enabledNoWindow.validate())

	// Enabled with a positive window is valid.
	enabledOK := base()
	enabledOK.RateLimitAuthRequests = 5
	enabledOK.RateLimitAuthWindow = time.Minute
	require.NoError(t, enabledOK.validate())
}

// clearKnownEnv unsets every environment variable the config reads so a test
// observes only defaults plus the values it explicitly sets. The previous value
// of each variable is captured and restored on cleanup. Variables must be truly
// unset (not merely empty) for envDefault to take effect.
func clearKnownEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"APP_ENV", "HTTP_PORT", "HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT", "SHUTDOWN_TIMEOUT", "MONGO_URI", "MONGO_DATABASE",
		"MONGO_TIMEOUT", "LOG_LEVEL",
		"JWT_SECRET", "ACCESS_TOKEN_TTL", "REFRESH_TTL_REMEMBER",
		"REFRESH_TTL_DEFAULT", "BCRYPT_COST", "COOKIE_DOMAIN", "COOKIE_SECURE",
		"CORS_ALLOWED_ORIGINS", "RATE_LIMIT_AUTH_REQUESTS", "RATE_LIMIT_AUTH_WINDOW",
	} {
		original, existed := os.LookupEnv(key)
		require.NoError(t, os.Unsetenv(key))
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(key, original)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}
