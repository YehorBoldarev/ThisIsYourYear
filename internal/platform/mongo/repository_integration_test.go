//go:build integration

package mongo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"kibou/thisisyouryear/internal/domain"
)

func TestRepository_InsertAndFind(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := NewRepository[sampleDoc, *sampleDoc](client.DB(), "samples_find", func() time.Time { return at })

	doc := &sampleDoc{Name: "alpha"}
	require.NoError(t, repo.Insert(ctx, doc))

	// Insert assigns identity and stamps both timestamps.
	require.False(t, doc.ID.IsZero())
	require.Equal(t, at, doc.CreatedAt)
	require.Equal(t, at, doc.UpdatedAt)

	got, err := repo.FindByID(ctx, HexFromID(doc.ID))
	require.NoError(t, err)
	require.Equal(t, "alpha", got.Name)
	require.Equal(t, doc.ID, got.ID)
}

func TestRepository_FindByID_NotFound(t *testing.T) {
	client := startMongo(t)
	repo := NewRepository[sampleDoc, *sampleDoc](client.DB(), "samples_nf", nil)

	_, err := repo.FindByID(context.Background(), HexFromID(newID()))
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestRepository_FindByID_InvalidHex(t *testing.T) {
	client := startMongo(t)
	repo := NewRepository[sampleDoc, *sampleDoc](client.DB(), "samples_badhex", nil)

	_, err := repo.FindByID(context.Background(), "nope")
	require.True(t, domain.IsValidation(err))
}

func TestRepository_Update(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := created.Add(48 * time.Hour)

	clock := created
	repo := NewRepository[sampleDoc, *sampleDoc](client.DB(), "samples_update", func() time.Time { return clock })

	doc := &sampleDoc{Name: "before"}
	require.NoError(t, repo.Insert(ctx, doc))

	clock = updated
	require.NoError(t, repo.Update(ctx, HexFromID(doc.ID), bson.M{"name": "after"}))

	got, err := repo.FindByID(ctx, HexFromID(doc.ID))
	require.NoError(t, err)
	require.Equal(t, "after", got.Name)
	require.Equal(t, created, got.CreatedAt, "createdAt must be preserved")
	require.Equal(t, updated, got.UpdatedAt, "updatedAt must advance")
}

func TestRepository_Update_NotFound(t *testing.T) {
	client := startMongo(t)
	repo := NewRepository[sampleDoc, *sampleDoc](client.DB(), "samples_update_nf", nil)

	err := repo.Update(context.Background(), HexFromID(newID()), bson.M{"name": "x"})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestRepository_Delete(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewRepository[sampleDoc, *sampleDoc](client.DB(), "samples_delete", nil)

	doc := &sampleDoc{Name: "doomed"}
	require.NoError(t, repo.Insert(ctx, doc))
	require.NoError(t, repo.Delete(ctx, HexFromID(doc.ID)))

	_, err := repo.FindByID(ctx, HexFromID(doc.ID))
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Deleting again is a not-found.
	require.ErrorIs(t, repo.Delete(ctx, HexFromID(doc.ID)), domain.ErrNotFound)
}

func TestRepository_List(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()
	repo := NewRepository[sampleDoc, *sampleDoc](client.DB(), "samples_list", nil)

	for _, name := range []string{"a", "b", "c"} {
		require.NoError(t, repo.Insert(ctx, &sampleDoc{Name: name}))
	}

	all, err := repo.List(ctx, bson.M{})
	require.NoError(t, err)
	require.Len(t, all, 3)

	one, err := repo.List(ctx, bson.M{"name": "b"})
	require.NoError(t, err)
	require.Len(t, one, 1)
	require.Equal(t, "b", one[0].Name)

	none, err := repo.List(ctx, bson.M{"name": "missing"})
	require.NoError(t, err)
	require.Empty(t, none, "no matches is an empty slice, not an error")
}

// TestEnsureIndexes_UniqueConstraint proves the index registry + EnsureIndexes
// path works end to end: a duplicate insert against a registered unique index is
// reported as domain.ErrConflict.
func TestEnsureIndexes_UniqueConstraint(t *testing.T) {
	client := startMongo(t)
	ctx := context.Background()

	client.RegisterIndexes("samples_unique", IndexSpec{
		Keys:   bson.D{{Key: "name", Value: 1}},
		Unique: true,
		Name:   "uniq_name",
	})
	require.NoError(t, client.EnsureIndexes(ctx))
	// Idempotent: ensuring again must not error.
	require.NoError(t, client.EnsureIndexes(ctx))

	repo := NewRepository[sampleDoc, *sampleDoc](client.DB(), "samples_unique", nil)
	require.NoError(t, repo.Insert(ctx, &sampleDoc{Name: "dup"}))

	err := repo.Insert(ctx, &sampleDoc{Name: "dup"})
	require.ErrorIs(t, err, domain.ErrConflict)
}

func newID() primitive.ObjectID { return primitive.NewObjectID() }
