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

func TestUserRepository_SetPasswordHashAndList(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewUserRepository(client, time.Now)
	require.NoError(t, client.EnsureIndexes(ctx))

	empty, err := repo.List(ctx)
	require.NoError(t, err)
	require.Empty(t, empty)

	first, err := domain.NewUser("first@example.com", "First", "old-hash")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, first))
	second, err := domain.NewUser("second@example.com", "Second", "hash-2")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, second))

	require.NoError(t, repo.SetPasswordHash(ctx, first.ID, "new-hash"))
	got, err := repo.ByID(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, "new-hash", got.PasswordHash)

	require.ErrorIs(t, repo.SetPasswordHash(ctx, "000000000000000000000000", "x"), domain.ErrNotFound)

	all, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, "first@example.com", all[0].Email, "oldest first")
	require.Equal(t, "second@example.com", all[1].Email)
}

// Account deletion (plan D3, Part C): delete-all is scoped to one user and
// leaves everyone else's data alone.
func TestAccountDeletion_ScopedToOneUser(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	users := NewUserRepository(client, time.Now)
	topics := NewTopicRepository(client, time.Now)
	goals := NewGoalRepository(client, time.Now)
	sessions := NewSessionRepository(client, time.Now)
	require.NoError(t, client.EnsureIndexes(ctx))

	seed := func(email string, nTopics, nGoals int) *domain.User {
		u, err := domain.NewUser(email, "U", "hash")
		require.NoError(t, err)
		require.NoError(t, users.Create(ctx, u))
		for i := 0; i < nTopics; i++ {
			topic, err := domain.NewTopic(u.ID, "T", "", 2027, "", "", "", nil)
			require.NoError(t, err)
			require.NoError(t, topics.Create(ctx, topic))
			for j := 0; j < nGoals; j++ {
				g, err := domain.NewGoal(u.ID, topic.ID, "G", "", false, nil)
				require.NoError(t, err)
				require.NoError(t, goals.Create(ctx, g))
			}
		}
		require.NoError(t, sessions.Create(ctx, &domain.Session{
			UserID: u.ID, RefreshTokenHash: "h-" + email, ExpiresAt: time.Now().Add(time.Hour),
		}))
		return u
	}
	alice := seed("alice@example.com", 2, 3)
	bob := seed("bob@example.com", 1, 1)

	n, err := goals.CountForUser(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, int64(6), n)
	n, err = topics.CountForUser(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	n, err = goals.DeleteAllForUser(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, int64(6), n)
	n, err = topics.DeleteAllForUser(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
	n, err = sessions.DeleteAllForUser(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	require.NoError(t, users.Delete(ctx, alice.ID))
	require.ErrorIs(t, users.Delete(ctx, alice.ID), domain.ErrNotFound)

	// Bob is untouched.
	n, err = goals.CountForUser(ctx, bob.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	n, err = sessions.CountForUser(ctx, bob.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	_, err = users.ByID(ctx, bob.ID)
	require.NoError(t, err)
}
