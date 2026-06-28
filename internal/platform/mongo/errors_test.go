package mongo

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"

	"kibou/thisisyouryear/internal/domain"
)

func TestTranslate(t *testing.T) {
	t.Run("nil passes through", func(t *testing.T) {
		require.NoError(t, translate(nil))
	})

	t.Run("no documents becomes not found", func(t *testing.T) {
		require.ErrorIs(t, translate(mongo.ErrNoDocuments), domain.ErrNotFound)
	})

	t.Run("duplicate key becomes conflict", func(t *testing.T) {
		dup := mongo.WriteException{
			WriteErrors: mongo.WriteErrors{{Code: 11000, Message: "E11000 duplicate key"}},
		}
		require.ErrorIs(t, translate(dup), domain.ErrConflict)
	})

	t.Run("unknown error is wrapped, not reclassified", func(t *testing.T) {
		sentinel := errors.New("connection reset")
		got := translate(sentinel)
		require.ErrorIs(t, got, sentinel)
		require.False(t, domain.IsNotFound(got))
		require.False(t, domain.IsConflict(got))
	})
}
