//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

func TestUserRepository_CreateAndFind(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	at := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	repo := NewUserRepository(client, fixedClock(at))
	require.NoError(t, client.EnsureIndexes(ctx))

	user, err := domain.NewUser("Alice@Example.com", "Alice", "hashed-pw")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, user))

	require.NotEmpty(t, user.ID, "Create assigns an id")
	require.Equal(t, at, user.CreatedAt)

	byEmail, err := repo.ByEmail(ctx, "  ALICE@example.com ")
	require.NoError(t, err)
	require.Equal(t, user.ID, byEmail.ID)
	require.Equal(t, "alice@example.com", byEmail.Email)
	require.Equal(t, "hashed-pw", byEmail.PasswordHash)

	byID, err := repo.ByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "alice@example.com", byID.Email)
}

func TestUserRepository_DuplicateEmailConflict(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewUserRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	first, err := domain.NewUser("dup@example.com", "First", "h1")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, first))

	second, err := domain.NewUser("DUP@example.com", "Second", "h2")
	require.NoError(t, err)
	err = repo.Create(ctx, second)
	require.ErrorIs(t, err, domain.ErrConflict, "the unique email index rejects the duplicate")
}

func TestUserRepository_NotFound(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewUserRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	_, err := repo.ByEmail(ctx, "nobody@example.com")
	require.ErrorIs(t, err, domain.ErrNotFound)
}
