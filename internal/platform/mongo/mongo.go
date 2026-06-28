// Package mongo owns the MongoDB connection lifecycle and index management for
// the application. It exposes a thin wrapper around the official driver so the
// rest of the codebase depends on a small, intention-revealing surface rather
// than the driver directly.
package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client bundles a connected Mongo client together with the handle to the
// application's database.
type Client struct {
	client *mongo.Client
	db     *mongo.Database
}

// Connect dials MongoDB at uri, verifies connectivity with a ping, and returns a
// ready-to-use Client scoped to the named database. The supplied timeout bounds
// both the initial connection and the verifying ping.
func Connect(ctx context.Context, uri, database string, timeout time.Duration) (*Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect to mongo: %w", err)
	}

	c := &Client{client: client, db: client.Database(database)}

	if err := c.Ping(connectCtx); err != nil {
		// Disconnect to avoid leaking the partially established client.
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("verify mongo connection: %w", err)
	}

	return c, nil
}

// DB returns the handle to the application's database.
func (c *Client) DB() *mongo.Database {
	return c.db
}

// Ping verifies the connection to MongoDB using the primary read preference.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("ping mongo: %w", err)
	}
	return nil
}

// EnsureIndexes creates any indexes the application relies on. It is idempotent
// and safe to call on every startup. Indexes are added here as collections are
// introduced by later steps; for now it is intentionally a no-op hook.
func (c *Client) EnsureIndexes(ctx context.Context) error {
	_ = ctx
	return nil
}

// Disconnect closes the underlying connection. It is safe to call during
// graceful shutdown; the provided context bounds how long disconnection waits.
func (c *Client) Disconnect(ctx context.Context) error {
	if err := c.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("disconnect mongo: %w", err)
	}
	return nil
}
