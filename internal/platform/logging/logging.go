// Package logging configures the application's structured logger.
//
// Production uses a JSON handler (machine-parseable, ready for log aggregation);
// development uses a human-readable text handler. The log level is driven by
// configuration.
package logging

import (
	"log/slog"
	"os"
)

// New builds a structured logger. When json is true a JSON handler is used,
// otherwise a text handler. The level string must be one of: debug, info, warn,
// error; any other value falls back to info.
func New(level string, json bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	if json {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
