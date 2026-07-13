//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"kibou/thisisyouryear/internal/domain"
)

func newTopic(t *testing.T, userID, title string, year int) *domain.Topic {
	t.Helper()
	topic, err := domain.NewTopic(userID, title, "", year, "", "", "", nil)
	require.NoError(t, err)
	return topic
}

func TestTopicRepository_CreateGetList(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewTopicRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	userID := primitive.NewObjectID().Hex()
	topic := newTopic(t, userID, "Get fit", 2026)
	require.NoError(t, repo.Create(ctx, topic))
	require.NotEmpty(t, topic.ID)

	got, err := repo.ByIDForUser(ctx, topic.ID, userID)
	require.NoError(t, err)
	require.Equal(t, "Get fit", got.Title)
	require.Equal(t, 2026, got.Year)

	// Another year is created; the 2026 filter excludes it.
	require.NoError(t, repo.Create(ctx, newTopic(t, userID, "Old goal", 2020)))
	y := 2026
	list, err := repo.ListForUser(ctx, userID, &y, false)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "Get fit", list[0].Title)
}

func TestTopicRepository_OwnershipIsolation(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewTopicRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	owner := primitive.NewObjectID().Hex()
	intruder := primitive.NewObjectID().Hex()
	topic := newTopic(t, owner, "Private", 2026)
	require.NoError(t, repo.Create(ctx, topic))

	// The intruder cannot read it.
	_, err := repo.ByIDForUser(ctx, topic.ID, intruder)
	require.ErrorIs(t, err, domain.ErrNotFound)

	// ...cannot update it (the owner's copy is untouched).
	newTitle := "Hijacked"
	err = repo.UpdateForUser(ctx, topic.ID, intruder, domain.TopicUpdate{Title: &newTitle})
	require.ErrorIs(t, err, domain.ErrNotFound)

	// ...and cannot delete it.
	err = repo.DeleteForUser(ctx, topic.ID, intruder)
	require.ErrorIs(t, err, domain.ErrNotFound)

	stillThere, err := repo.ByIDForUser(ctx, topic.ID, owner)
	require.NoError(t, err)
	require.Equal(t, "Private", stillThere.Title, "the owner's topic is unchanged")

	// The intruder's list is empty.
	list, err := repo.ListForUser(ctx, intruder, nil, true)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestTopicRepository_UpdateAndArchive(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewTopicRepository(client, nil)
	require.NoError(t, client.EnsureIndexes(ctx))

	userID := primitive.NewObjectID().Hex()
	topic := newTopic(t, userID, "Before", 2026)
	require.NoError(t, repo.Create(ctx, topic))

	newTitle := "After"
	archived := true
	require.NoError(t, repo.UpdateForUser(ctx, topic.ID, userID, domain.TopicUpdate{
		Title:    &newTitle,
		Archived: &archived,
	}))

	got, err := repo.ByIDForUser(ctx, topic.ID, userID)
	require.NoError(t, err)
	require.Equal(t, "After", got.Title)
	require.True(t, got.IsArchived())

	// Archived topics are excluded by default, included on request.
	active, err := repo.ListForUser(ctx, userID, nil, false)
	require.NoError(t, err)
	require.Empty(t, active)

	all, err := repo.ListForUser(ctx, userID, nil, true)
	require.NoError(t, err)
	require.Len(t, all, 1)

	// Unarchiving clears the flag.
	unarchive := false
	require.NoError(t, repo.UpdateForUser(ctx, topic.ID, userID, domain.TopicUpdate{Archived: &unarchive}))
	got, err = repo.ByIDForUser(ctx, topic.ID, userID)
	require.NoError(t, err)
	require.False(t, got.IsArchived())
}
