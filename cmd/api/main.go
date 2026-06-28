// Command api is the entrypoint for the ThisIsYourYear backend service.
//
// It loads configuration, builds the logger, connects to MongoDB, constructs the
// HTTP server, and runs until interrupted, at which point it shuts everything
// down gracefully.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"kibou/thisisyouryear/internal/config"
	"kibou/thisisyouryear/internal/platform/logging"
	"kibou/thisisyouryear/internal/platform/mongo"
	transporthttp "kibou/thisisyouryear/internal/transport/http"
)

func main() {
	if err := run(); err != nil {
		// The logger may not exist yet if configuration failed, so fall back to
		// the default slog logger for this terminal error.
		slog.Error("fatal startup error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel, cfg.IsProduction())
	logger.Info("configuration loaded",
		slog.String("app_env", string(cfg.AppEnv)),
		slog.Int("http_port", cfg.HTTPPort),
		slog.String("mongo_database", cfg.MongoDatabase),
	)

	// Root context cancelled on SIGINT/SIGTERM drives graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mongoClient, err := mongo.Connect(ctx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoTimeout)
	if err != nil {
		return err
	}
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if derr := mongoClient.Disconnect(disconnectCtx); derr != nil {
			logger.Error("mongo disconnect failed", slog.Any("error", derr))
		}
	}()
	logger.Info("connected to mongo")

	if err := mongoClient.EnsureIndexes(ctx); err != nil {
		return err
	}

	router := transporthttp.NewRouter(transporthttp.RouterDeps{
		Logger: logger,
		Pinger: mongoClient,
	})

	server := transporthttp.NewServer(transporthttp.ServerConfig{
		Port:            cfg.HTTPPort,
		ReadTimeout:     cfg.HTTPReadTimeout,
		WriteTimeout:    cfg.HTTPWriteTimeout,
		IdleTimeout:     cfg.HTTPIdleTimeout,
		ShutdownTimeout: cfg.ShutdownTimeout,
	}, router, logger)

	return server.Run(ctx)
}
