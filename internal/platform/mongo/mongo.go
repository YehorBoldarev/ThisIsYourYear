// Package mongo owns the MongoDB connection lifecycle and index management for
// the application. It exposes a thin wrapper around the official driver so the
// rest of the codebase depends on a small, intention-revealing surface rather
// than the driver directly.
package mongo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client bundles a connected Mongo client together with the handle to the
// application's database and a registry of indexes to ensure on startup.
type Client struct {
	client *mongo.Client
	db     *mongo.Database

	mu      sync.Mutex
	indexes map[string][]IndexSpec // collection name -> specs
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

	c := &Client{
		client:  client,
		db:      client.Database(database),
		indexes: make(map[string][]IndexSpec),
	}

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

// RegisterIndexes records index specs for a collection to be created by
// EnsureIndexes. Concrete repositories call this at construction time, before the
// application invokes EnsureIndexes during startup. It is safe for concurrent use.
func (c *Client) RegisterIndexes(collection string, specs ...IndexSpec) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.indexes[collection] = append(c.indexes[collection], specs...)
}

// EnsureIndexes creates every registered index. Creating an index that already
// exists with the same definition is a no-op, so this is idempotent and safe to
// call on every startup.
func (c *Client) EnsureIndexes(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for collection, specs := range c.indexes {
		if len(specs) == 0 {
			continue
		}
		models := make([]mongo.IndexModel, 0, len(specs))
		for _, spec := range specs {
			models = append(models, spec.model())
		}
		if _, err := c.db.Collection(collection).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("ensure indexes for collection %q: %w", collection, err)
		}
	}
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
