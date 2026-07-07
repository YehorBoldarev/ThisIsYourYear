//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"kibou/thisisyouryear/internal/domain"
)

func newGoal(t *testing.T, userID, topicID, title string, order int) *domain.Goal {
	t.Helper()
	g, err := domain.NewGoal(userID, topicID, title, "")
	require.NoError(t, err)
	g.Order = order
	return g
}

func TestGoalRepository_CreateListOrder(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewGoalRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	userID := primitive.NewObjectID().Hex()
	topicID := primitive.NewObjectID().Hex()

	// Insert out of order; listing sorts by order.
	require.NoError(t, repo.Create(ctx, newGoal(t, userID, topicID, "second", 1)))
	require.NoError(t, repo.Create(ctx, newGoal(t, userID, topicID, "first", 0)))

	goals, err := repo.ListForTopic(ctx, userID, topicID)
	require.NoError(t, err)
	require.Len(t, goals, 2)
	require.Equal(t, "first", goals[0].Title)
	require.Equal(t, "second", goals[1].Title)

	count, err := repo.CountForTopic(ctx, userID, topicID)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestGoalRepository_MarkDoneClearsAndSets(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewGoalRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	userID := primitive.NewObjectID().Hex()
	topicID := primitive.NewObjectID().Hex()
	g := newGoal(t, userID, topicID, "Run", 0)
	require.NoError(t, repo.Create(ctx, g))

	done := true
	require.NoError(t, repo.UpdateForUser(ctx, g.ID, userID, domain.GoalUpdate{Done: &done}))
	got, err := repo.ByIDForUser(ctx, g.ID, userID)
	require.NoError(t, err)
	require.True(t, got.IsAchieved())
	require.NotNil(t, got.CompletedAt)

	undone := false
	require.NoError(t, repo.UpdateForUser(ctx, g.ID, userID, domain.GoalUpdate{Done: &undone}))
	got, err = repo.ByIDForUser(ctx, g.ID, userID)
	require.NoError(t, err)
	require.Equal(t, domain.GoalStatusTodo, got.Status)
	require.Nil(t, got.CompletedAt, "undoing clears completedAt")
}

func TestGoalRepository_CountsByTopics(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewGoalRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	userID := primitive.NewObjectID().Hex()
	topicA := primitive.NewObjectID().Hex()
	topicB := primitive.NewObjectID().Hex()
	empty := primitive.NewObjectID().Hex()

	// Topic A: 3 goals, 2 done. Topic B: 1 goal, 0 done.
	for i, done := range []bool{true, true, false} {
		g := newGoal(t, userID, topicA, "a", i)
		if done {
			g.Status = domain.GoalStatusDone
		}
		require.NoError(t, repo.Create(ctx, g))
	}
	require.NoError(t, repo.Create(ctx, newGoal(t, userID, topicB, "b", 0)))

	counts, err := repo.CountsByTopics(ctx, userID, []string{topicA, topicB, empty})
	require.NoError(t, err)

	require.Equal(t, 3, counts[topicA].Total)
	require.Equal(t, 2, counts[topicA].Completed)
	require.InDelta(t, 2.0/3.0, counts[topicA].Ratio, 1e-9)
	require.Equal(t, 1, counts[topicB].Total)
	require.Equal(t, 0, counts[topicB].Completed)
	_, present := counts[empty]
	require.False(t, present, "a topic with no goals is absent from the map")
}

func TestGoalRepository_DeleteByTopic(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewGoalRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	userID := primitive.NewObjectID().Hex()
	topicID := primitive.NewObjectID().Hex()
	other := primitive.NewObjectID().Hex()
	require.NoError(t, repo.Create(ctx, newGoal(t, userID, topicID, "g1", 0)))
	require.NoError(t, repo.Create(ctx, newGoal(t, userID, topicID, "g2", 1)))
	require.NoError(t, repo.Create(ctx, newGoal(t, userID, other, "keep", 0)))

	require.NoError(t, repo.DeleteByTopic(ctx, userID, topicID))

	gone, err := repo.ListForTopic(ctx, userID, topicID)
	require.NoError(t, err)
	require.Empty(t, gone)

	kept, err := repo.ListForTopic(ctx, userID, other)
	require.NoError(t, err)
	require.Len(t, kept, 1, "goals under other topics are untouched")
}

func TestGoalRepository_OwnershipIsolation(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewGoalRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	owner := primitive.NewObjectID().Hex()
	intruder := primitive.NewObjectID().Hex()
	topicID := primitive.NewObjectID().Hex()
	g := newGoal(t, owner, topicID, "Private", 0)
	require.NoError(t, repo.Create(ctx, g))

	_, err := repo.ByIDForUser(ctx, g.ID, intruder)
	require.ErrorIs(t, err, domain.ErrNotFound)

	title := "hijack"
	require.ErrorIs(t, repo.UpdateForUser(ctx, g.ID, intruder, domain.GoalUpdate{Title: &title}), domain.ErrNotFound)
	require.ErrorIs(t, repo.DeleteForUser(ctx, g.ID, intruder), domain.ErrNotFound)

	// Owner's goal is intact.
	got, err := repo.ByIDForUser(ctx, g.ID, owner)
	require.NoError(t, err)
	require.Equal(t, "Private", got.Title)
}
