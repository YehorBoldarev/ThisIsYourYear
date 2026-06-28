package mongo

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"kibou/thisisyouryear/internal/domain"
)

// Document is embedded by every persisted entity. It carries the Mongo identity
// and audit timestamps so individual entities need not redeclare them, and so the
// generic repository can manage them uniformly.
//
// Embed it by value with the inline tag, e.g.:
//
//	type User struct {
//	    mongo.Document `bson:",inline"`
//	    Email string   `bson:"email"`
//	}
type Document struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	CreatedAt time.Time          `bson:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"`
}

// The unexported accessors below let the generic repository read and stamp the
// embedded fields without reflection. They are promoted onto any *T that embeds
// Document, which is how *T satisfies the documentPtr constraint — including for
// entities declared in other packages, since the promoted methods retain their
// identity in this package.

func (d *Document) getID() primitive.ObjectID   { return d.ID }
func (d *Document) setID(id primitive.ObjectID) { d.ID = id }
func (d *Document) setCreatedAt(t time.Time)    { d.CreatedAt = t }
func (d *Document) setUpdatedAt(t time.Time)    { d.UpdatedAt = t }

// documentPtr is satisfied by *T for any entity T that embeds Document. It is the
// behavioral contract the generic Repository requires of its element type.
type documentPtr interface {
	getID() primitive.ObjectID
	setID(primitive.ObjectID)
	setCreatedAt(time.Time)
	setUpdatedAt(time.Time)
}

// Clock returns the current time. It is injected into repositories so timestamp
// behavior is deterministic in tests; production uses time.Now.
type Clock func() time.Time

// HexFromID renders an ObjectID as the hex string used at the API boundary.
func HexFromID(id primitive.ObjectID) string { return id.Hex() }

// IDFromHex parses an API-facing hex string into an ObjectID. An invalid hex
// value is a client error, so it is reported as domain.ErrValidation.
func IDFromHex(hex string) (primitive.ObjectID, error) {
	id, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("%w: invalid id %q", domain.ErrValidation, hex)
	}
	return id, nil
}
