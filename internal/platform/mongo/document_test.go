package mongo

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"kibou/thisisyouryear/internal/domain"
)

func TestIDRoundTrip(t *testing.T) {
	id := primitive.NewObjectID()
	hex := HexFromID(id)

	got, err := IDFromHex(hex)
	require.NoError(t, err)
	require.Equal(t, id, got)
}

func TestIDFromHex_Invalid(t *testing.T) {
	_, err := IDFromHex("not-a-valid-object-id")
	require.Error(t, err)
	require.True(t, domain.IsValidation(err), "invalid id must be a validation error")
}

// sampleDoc is a test-only entity proving an arbitrary struct that embeds
// Document gains the accessors the generic repository requires.
type sampleDoc struct {
	Document `bson:",inline"`
	Name     string `bson:"name"`
}

func TestDocumentAccessors(t *testing.T) {
	// *sampleDoc must satisfy documentPtr purely via the embedded Document.
	var p documentPtr = &sampleDoc{}

	id := primitive.NewObjectID()
	p.setID(id)
	require.Equal(t, id, p.getID())
}
