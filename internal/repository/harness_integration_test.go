//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"

	"kibou/thisisyouryear/internal/platform/mongo"
)

// startMongo launches a throwaway MongoDB container and returns a connected
// Client scoped to a fresh database, torn down automatically when the test ends.
// Requires Docker; compiled only under the `integration` build tag.
func startMongo(t *testing.T) *mongo.Client {
	t.Helper()
	ctx := context.Background()

	container, err := mongodb.Run(ctx, "mongo:7")
	require.NoError(t, err, "start mongo container")
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	uri, err := container.ConnectionString(ctx)
	require.NoError(t, err, "mongo connection string")

	client, err := mongo.Connect(ctx, uri, "thisisyouryear_test", 30*time.Second)
	require.NoError(t, err, "connect to mongo container")
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	return client
}

// fixedClock returns a deterministic Clock for timestamp assertions.
func fixedClock(at time.Time) mongo.Clock { return func() time.Time { return at } }
