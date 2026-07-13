package repository

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/platform/mongo"
)

const topicsCollection = "topics"

type topicDoc struct {
	mongo.Document `bson:",inline"`
	UserID         primitive.ObjectID `bson:"userId"`
	Title          string             `bson:"title"`
	Description    string             `bson:"description"`
	Year           int                `bson:"year"`
	Color          string             `bson:"color"`
	Icon           string             `bson:"icon"`
	Category       string             `bson:"category,omitempty"`
	TargetDate     *time.Time         `bson:"targetDate,omitempty"`
	ArchivedAt     *time.Time         `bson:"archivedAt,omitempty"`
}

func (d *topicDoc) toDomain() *domain.Topic {
	return &domain.Topic{
		ID:          mongo.HexFromID(d.ID),
		UserID:      mongo.HexFromID(d.UserID),
		Title:       d.Title,
		Description: d.Description,
		Year:        d.Year,
		Color:       d.Color,
		Icon:        d.Icon,
		Category:    d.Category,
		TargetDate:  d.TargetDate,
		ArchivedAt:  d.ArchivedAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// TopicRepository persists topics. Every read and write is scoped by userId, so
// one user can never reach another user's topics.
type TopicRepository struct {
	repo *mongo.Repository[topicDoc, *topicDoc]
}

// NewTopicRepository builds the repository and registers the topic indexes: a
// compound index on (userId, year) for the year-scoped list, and (userId,
// archivedAt) for the active/archived filter.
func NewTopicRepository(client *mongo.Client, clock mongo.Clock) *TopicRepository {
	client.RegisterIndexes(topicsCollection,
		mongo.IndexSpec{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "year", Value: 1}},
			Name: "idx_userId_year",
		},
		mongo.IndexSpec{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "archivedAt", Value: 1}},
			Name: "idx_userId_archivedAt",
		},
	)
	return &TopicRepository{
		repo: mongo.NewRepository[topicDoc, *topicDoc](client.DB(), topicsCollection, clock),
	}
}

// Create persists a new topic. On success the topic's ID and timestamps are
// populated from the stored document.
func (r *TopicRepository) Create(ctx context.Context, t *domain.Topic) error {
	uid, err := mongo.IDFromHex(t.UserID)
	if err != nil {
		return err
	}
	doc := &topicDoc{
		UserID:      uid,
		Title:       t.Title,
		Description: t.Description,
		Year:        t.Year,
		Color:       t.Color,
		Icon:        t.Icon,
		Category:    t.Category,
		TargetDate:  t.TargetDate,
		ArchivedAt:  t.ArchivedAt,
	}
	if err := r.repo.Insert(ctx, doc); err != nil {
		return err
	}
	t.ID = mongo.HexFromID(doc.ID)
	t.CreatedAt = doc.CreatedAt
	t.UpdatedAt = doc.UpdatedAt
	return nil
}

// ByIDForUser returns the caller's topic with the given id, or domain.ErrNotFound
// if it does not exist or belongs to another user (existence is not leaked).
func (r *TopicRepository) ByIDForUser(ctx context.Context, id, userID string) (*domain.Topic, error) {
	oid, err := mongo.IDFromHex(id)
	if err != nil {
		return nil, err
	}
	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return nil, err
	}
	doc, err := r.repo.FindOne(ctx, bson.M{"_id": oid, "userId": uid})
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// ListForUser returns the caller's topics, optionally filtered to a year and
// (by default) excluding archived topics. Results are ordered oldest-first.
func (r *TopicRepository) ListForUser(ctx context.Context, userID string, year *int, includeArchived bool) ([]domain.Topic, error) {
	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"userId": uid}
	if year != nil {
		filter["year"] = *year
	}
	if !includeArchived {
		filter["archivedAt"] = nil
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})
	docs, err := r.repo.List(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	topics := make([]domain.Topic, 0, len(docs))
	for i := range docs {
		topics = append(topics, *docs[i].toDomain())
	}
	return topics, nil
}

// UpdateForUser applies a partial update to the caller's topic. A missing or
// non-owned topic yields domain.ErrNotFound.
func (r *TopicRepository) UpdateForUser(ctx context.Context, id, userID string, upd domain.TopicUpdate) error {
	oid, err := mongo.IDFromHex(id)
	if err != nil {
		return err
	}
	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return err
	}

	fields := topicUpdateFields(upd)
	matched, err := r.repo.UpdateBy(ctx, bson.M{"_id": oid, "userId": uid}, fields)
	if err != nil {
		return err
	}
	if matched == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteForUser removes the caller's topic. A missing or non-owned topic yields
// domain.ErrNotFound.
func (r *TopicRepository) DeleteForUser(ctx context.Context, id, userID string) error {
	oid, err := mongo.IDFromHex(id)
	if err != nil {
		return err
	}
	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return err
	}
	deleted, err := r.repo.DeleteBy(ctx, bson.M{"_id": oid, "userId": uid})
	if err != nil {
		return err
	}
	if deleted == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// topicUpdateFields turns a domain TopicUpdate into the bson $set fields. Only
// supplied fields are included; archiving maps the Archived flag to setting or
// clearing archivedAt.
func topicUpdateFields(u domain.TopicUpdate) bson.M {
	fields := bson.M{}
	if u.Title != nil {
		fields["title"] = strings.TrimSpace(*u.Title)
	}
	if u.Description != nil {
		fields["description"] = strings.TrimSpace(*u.Description)
	}
	if u.Year != nil {
		fields["year"] = *u.Year
	}
	if u.Color != nil {
		fields["color"] = strings.TrimSpace(*u.Color)
	}
	if u.Icon != nil {
		fields["icon"] = strings.TrimSpace(*u.Icon)
	}
	if u.Category != nil {
		fields["category"] = strings.TrimSpace(*u.Category)
	}
	// A present target-date update sets the date, or clears it (nil -> null).
	if td := u.NormalizedTargetDate(); td.Present {
		fields["targetDate"] = td.Value
	}
	if u.Archived != nil {
		if *u.Archived {
			fields["archivedAt"] = time.Now().UTC()
		} else {
			fields["archivedAt"] = nil
		}
	}
	return fields
}
