package mongo

import (
	"context"
	"maps"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"kibou/thisisyouryear/internal/domain"
)

// Repository is a reusable, type-safe persistence base for a single collection of
// entity T. Concrete repositories embed or wrap it and add only entity-specific
// queries, so adding a new persisted resource is a thin, boilerplate-free wrapper.
//
// The PT type parameter is the pointer type *T; the constraint requires it to
// satisfy documentPtr, which holds for any T that embeds Document. Both type
// arguments are given explicitly at construction, e.g.:
//
//	repo := mongo.NewRepository[User, *User](db, "users", nil)
type Repository[T any, PT interface {
	*T
	documentPtr
}] struct {
	coll  *mongo.Collection
	clock Clock
}

// NewRepository builds a Repository over the named collection. A nil clock
// defaults to time.Now.
func NewRepository[T any, PT interface {
	*T
	documentPtr
}](db *mongo.Database, collection string, clock Clock) *Repository[T, PT] {
	if clock == nil {
		clock = time.Now
	}
	return &Repository[T, PT]{coll: db.Collection(collection), clock: clock}
}

// Collection exposes the underlying collection for entity-specific operations a
// concrete repository may need beyond the generic CRUD methods.
func (r *Repository[T, PT]) Collection() *mongo.Collection { return r.coll }

// Insert persists entity, assigning a new ID if it has none and stamping the
// created/updated timestamps. The entity is mutated in place so the caller can
// read back the generated identity and times.
func (r *Repository[T, PT]) Insert(ctx context.Context, entity *T) error {
	p := PT(entity)
	now := r.clock().UTC()
	if p.getID().IsZero() {
		p.setID(primitive.NewObjectID())
	}
	p.setCreatedAt(now)
	p.setUpdatedAt(now)

	if _, err := r.coll.InsertOne(ctx, entity); err != nil {
		return translate(err)
	}
	return nil
}

// FindByID returns the entity with the given hex id, or domain.ErrNotFound.
func (r *Repository[T, PT]) FindByID(ctx context.Context, id string) (*T, error) {
	oid, err := IDFromHex(id)
	if err != nil {
		return nil, err
	}
	return r.FindOne(ctx, bson.M{"_id": oid})
}

// FindOne returns the first entity matching filter, or domain.ErrNotFound.
func (r *Repository[T, PT]) FindOne(ctx context.Context, filter bson.M) (*T, error) {
	var out T
	if err := r.coll.FindOne(ctx, filter).Decode(&out); err != nil {
		return nil, translate(err)
	}
	return &out, nil
}

// Update applies fields as a $set on the document with the given hex id and bumps
// updatedAt. fields is a map of field name to new value (not raw update
// operators). A missing document yields domain.ErrNotFound.
func (r *Repository[T, PT]) Update(ctx context.Context, id string, fields bson.M) error {
	oid, err := IDFromHex(id)
	if err != nil {
		return err
	}
	matched, err := r.UpdateBy(ctx, bson.M{"_id": oid}, fields)
	if err != nil {
		return err
	}
	if matched == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UpdateBy applies fields as a $set on the single document matching filter and
// bumps updatedAt, returning the matched count. Unlike Update it does not treat
// a zero match as an error: callers can match on extra fields (e.g. userId) to
// enforce ownership and decide for themselves what a miss means.
func (r *Repository[T, PT]) UpdateBy(ctx context.Context, filter bson.M, fields bson.M) (int64, error) {
	set := make(bson.M, len(fields)+1)
	maps.Copy(set, fields)
	set["updatedAt"] = r.clock().UTC()

	res, err := r.coll.UpdateOne(ctx, filter, bson.M{"$set": set})
	if err != nil {
		return 0, translate(err)
	}
	return res.MatchedCount, nil
}

// UpdateMany applies fields as a $set on every document matching filter and
// bumps updatedAt on each. It returns the number of documents modified; matching
// zero documents is not an error. Use it for bulk state changes such as revoking
// all of a user's sessions.
func (r *Repository[T, PT]) UpdateMany(ctx context.Context, filter bson.M, fields bson.M) (int64, error) {
	set := make(bson.M, len(fields)+1)
	maps.Copy(set, fields)
	set["updatedAt"] = r.clock().UTC()

	res, err := r.coll.UpdateMany(ctx, filter, bson.M{"$set": set})
	if err != nil {
		return 0, translate(err)
	}
	return res.ModifiedCount, nil
}

// Delete removes the document with the given hex id (hard delete). A missing
// document yields domain.ErrNotFound.
func (r *Repository[T, PT]) Delete(ctx context.Context, id string) error {
	oid, err := IDFromHex(id)
	if err != nil {
		return err
	}
	deleted, err := r.DeleteBy(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if deleted == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteBy removes the single document matching filter, returning the deleted
// count. Like UpdateBy it leaves the zero-match policy to the caller, enabling
// ownership-scoped deletes via an extra filter field.
func (r *Repository[T, PT]) DeleteBy(ctx context.Context, filter bson.M) (int64, error) {
	res, err := r.coll.DeleteOne(ctx, filter)
	if err != nil {
		return 0, translate(err)
	}
	return res.DeletedCount, nil
}

// DeleteManyBy removes every document matching filter, returning the deleted
// count. Matching zero documents is not an error. Used for cascades such as
// removing all goals under a deleted topic.
func (r *Repository[T, PT]) DeleteManyBy(ctx context.Context, filter bson.M) (int64, error) {
	res, err := r.coll.DeleteMany(ctx, filter)
	if err != nil {
		return 0, translate(err)
	}
	return res.DeletedCount, nil
}

// CountBy returns the number of documents matching filter.
func (r *Repository[T, PT]) CountBy(ctx context.Context, filter bson.M) (int64, error) {
	n, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return 0, translate(err)
	}
	return n, nil
}

// Aggregate runs an aggregation pipeline and decodes all result documents into
// out (a pointer to a slice). It centralizes cursor handling and error
// translation so concrete repositories can express aggregations declaratively.
func (r *Repository[T, PT]) Aggregate(ctx context.Context, pipeline any, out any) error {
	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return translate(err)
	}
	defer func() { _ = cur.Close(ctx) }()
	if err := cur.All(ctx, out); err != nil {
		return translate(err)
	}
	return nil
}

// List returns all entities matching filter. Pass FindOptions for sorting,
// pagination, or projection. An empty result is not an error.
func (r *Repository[T, PT]) List(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]T, error) {
	cur, err := r.coll.Find(ctx, filter, opts...)
	if err != nil {
		return nil, translate(err)
	}
	defer func() { _ = cur.Close(ctx) }()

	out := make([]T, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, translate(err)
	}
	return out, nil
}
