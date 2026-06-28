package mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// IndexSpec is a storage-agnostic description of an index a collection relies on.
// Concrete repositories declare their specs and register them with the Client;
// EnsureIndexes applies them idempotently on startup.
type IndexSpec struct {
	// Keys lists the indexed fields in order, each mapped to 1 (ascending) or
	// -1 (descending), e.g. bson.D{{Key: "email", Value: 1}}.
	Keys bson.D
	// Unique enforces a uniqueness constraint across the keyed fields.
	Unique bool
	// TTL, when set, expires documents this long after the indexed time field.
	// Only valid on a single date-typed key.
	TTL *time.Duration
	// Name overrides the auto-generated index name; optional but recommended for
	// stable, recognizable names.
	Name string
}

// model converts the spec into the driver's index model.
func (s IndexSpec) model() mongo.IndexModel {
	opts := options.Index()
	if s.Unique {
		opts.SetUnique(true)
	}
	if s.Name != "" {
		opts.SetName(s.Name)
	}
	if s.TTL != nil {
		opts.SetExpireAfterSeconds(int32(s.TTL.Seconds()))
	}
	return mongo.IndexModel{Keys: s.Keys, Options: opts}
}
