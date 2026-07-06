//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"kibou/thisisyouryear/internal/domain"
)

func newSession(userID, hash string) *domain.Session {
	return &domain.Session{
		UserID:           userID,
		RefreshTokenHash: hash,
		RememberMe:       true,
		UserAgent:        "test-agent",
		IP:               "127.0.0.1",
		ExpiresAt:        time.Now().Add(time.Hour).UTC(),
	}
}

func TestSessionRepository_CreateAndLookup(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewSessionRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	userID := primitive.NewObjectID().Hex()
	s := newSession(userID, "hash-abc")
	require.NoError(t, repo.Create(ctx, s))
	require.NotEmpty(t, s.ID)

	got, err := repo.ByTokenHash(ctx, "hash-abc")
	require.NoError(t, err)
	require.Equal(t, s.ID, got.ID)
	require.Equal(t, userID, got.UserID)
	require.True(t, got.IsActive(time.Now()))
}

func TestSessionRepository_Revoke(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewSessionRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	s := newSession(primitive.NewObjectID().Hex(), "hash-revoke")
	require.NoError(t, repo.Create(ctx, s))

	require.NoError(t, repo.Revoke(ctx, s.ID, time.Now().UTC()))

	got, err := repo.ByTokenHash(ctx, "hash-revoke")
	require.NoError(t, err)
	require.NotNil(t, got.RevokedAt)
	require.False(t, got.IsActive(time.Now()), "a revoked session is inactive")
}

func TestSessionRepository_RevokeAllForUser(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewSessionRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	userID := primitive.NewObjectID().Hex()
	require.NoError(t, repo.Create(ctx, newSession(userID, "h1")))
	require.NoError(t, repo.Create(ctx, newSession(userID, "h2")))
	// A different user's session must be untouched.
	other := primitive.NewObjectID().Hex()
	require.NoError(t, repo.Create(ctx, newSession(other, "h3")))

	require.NoError(t, repo.RevokeAllForUser(ctx, userID, time.Now().UTC()))

	for _, h := range []string{"h1", "h2"} {
		got, err := repo.ByTokenHash(ctx, h)
		require.NoError(t, err)
		require.NotNil(t, got.RevokedAt, "session %s should be revoked", h)
	}
	got, err := repo.ByTokenHash(ctx, "h3")
	require.NoError(t, err)
	require.Nil(t, got.RevokedAt, "other user's session must remain active")
}

func TestSessionRepository_DuplicateTokenHashConflict(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewSessionRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	userID := primitive.NewObjectID().Hex()
	require.NoError(t, repo.Create(ctx, newSession(userID, "same-hash")))
	err := repo.Create(ctx, newSession(userID, "same-hash"))
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestSessionRepository_TTLIndexExists(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	_ = NewSessionRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	cursor, err := client.DB().Collection(sessionsCollection).Indexes().List(ctx)
	require.NoError(t, err)
	var indexes []bson.M
	require.NoError(t, cursor.All(ctx, &indexes))

	var found bool
	for _, idx := range indexes {
		if _, ok := idx["expireAfterSeconds"]; ok {
			key, _ := idx["key"].(bson.M)
			if _, keyed := key["expiresAt"]; keyed {
				found = true
			}
		}
	}
	require.True(t, found, "a TTL index on expiresAt must be present")
}
