// Package config loads and validates application configuration from the
// environment, following 12-factor conventions. No configuration values are
// baked into the binary or images; everything comes from environment variables
// (optionally seeded from a local .env file during development).
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Environment enumerates the supported runtime environments.
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvProduction  Environment = "production"
)

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

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
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
