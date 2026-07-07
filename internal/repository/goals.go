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

const goalsCollection = "goals"

type goalDoc struct {
	mongo.Document `bson:",inline"`
	TopicID        primitive.ObjectID `bson:"topicId"`
	UserID         primitive.ObjectID `bson:"userId"`
	Title          string             `bson:"title"`
	Notes          string             `bson:"notes"`
	Status         string             `bson:"status"`
	CompletedAt    *time.Time         `bson:"completedAt,omitempty"`
	Order          int                `bson:"order"`
}

func (d *goalDoc) toDomain() *domain.Goal {
	return &domain.Goal{
		ID:          mongo.HexFromID(d.ID),
		TopicID:     mongo.HexFromID(d.TopicID),
		UserID:      mongo.HexFromID(d.UserID),
		Title:       d.Title,
		Notes:       d.Notes,
		Status:      domain.GoalStatus(d.Status),
		CompletedAt: d.CompletedAt,
		Order:       d.Order,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// GoalRepository persists goals. Every read and write is scoped by userId, so a
// user can only ever reach goals they own.
type GoalRepository struct {
	repo *mongo.Repository[goalDoc, *goalDoc]
}

// NewGoalRepository builds the repository and registers the compound index that
// serves ordered listing within a topic, progress counts, ownership checks, and
// the cascade delete.
func NewGoalRepository(client *mongo.Client, clock mongo.Clock) *GoalRepository {
	client.RegisterIndexes(goalsCollection,
		mongo.IndexSpec{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "topicId", Value: 1}, {Key: "order", Value: 1}},
			Name: "idx_userId_topicId_order",
		},
	)
	return &GoalRepository{
		repo: mongo.NewRepository[goalDoc, *goalDoc](client.DB(), goalsCollection, clock),
	}
}

// Create persists a new goal. On success the goal's ID and timestamps are
// populated from the stored document.
func (r *GoalRepository) Create(ctx context.Context, g *domain.Goal) error {
	tid, err := mongo.IDFromHex(g.TopicID)
	if err != nil {
		return err
	}
	uid, err := mongo.IDFromHex(g.UserID)
	if err != nil {
		return err
	}
	doc := &goalDoc{
		TopicID:     tid,
		UserID:      uid,
		Title:       g.Title,
		Notes:       g.Notes,
		Status:      string(g.Status),
		CompletedAt: g.CompletedAt,
		Order:       g.Order,
	}
	if err := r.repo.Insert(ctx, doc); err != nil {
		return err
	}
	g.ID = mongo.HexFromID(doc.ID)
	g.CreatedAt = doc.CreatedAt
	g.UpdatedAt = doc.UpdatedAt
	return nil
}

// ByIDForUser returns the caller's goal, or domain.ErrNotFound if it is missing
// or owned by another user.
func (r *GoalRepository) ByIDForUser(ctx context.Context, id, userID string) (*domain.Goal, error) {
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

// ListForTopic returns the caller's goals under a topic, ordered by order then
// creation time.
func (r *GoalRepository) ListForTopic(ctx context.Context, userID, topicID string) ([]domain.Goal, error) {
	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return nil, err
	}
	tid, err := mongo.IDFromHex(topicID)
	if err != nil {
		return nil, err
	}
	opts := options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "createdAt", Value: 1}})
	docs, err := r.repo.List(ctx, bson.M{"userId": uid, "topicId": tid}, opts)
	if err != nil {
		return nil, err
	}
	goals := make([]domain.Goal, 0, len(docs))
	for i := range docs {
		goals = append(goals, *docs[i].toDomain())
	}
	return goals, nil
}

// UpdateForUser applies a partial update to the caller's goal. A missing or
// non-owned goal yields domain.ErrNotFound.
func (r *GoalRepository) UpdateForUser(ctx context.Context, id, userID string, upd domain.GoalUpdate) error {
	oid, err := mongo.IDFromHex(id)
	if err != nil {
		return err
	}
	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return err
	}
	matched, err := r.repo.UpdateBy(ctx, bson.M{"_id": oid, "userId": uid}, goalUpdateFields(upd))
	if err != nil {
		return err
	}
	if matched == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteForUser removes the caller's goal, or domain.ErrNotFound.
func (r *GoalRepository) DeleteForUser(ctx context.Context, id, userID string) error {
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

// DeleteByTopic removes every goal under a topic for a user. It is used to
// cascade a topic deletion; removing zero goals is not an error.
func (r *GoalRepository) DeleteByTopic(ctx context.Context, userID, topicID string) error {
	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return err
	}
	tid, err := mongo.IDFromHex(topicID)
	if err != nil {
		return err
	}
	_, err = r.repo.DeleteManyBy(ctx, bson.M{"userId": uid, "topicId": tid})
	return err
}

// CountForTopic returns how many goals a user has under a topic, used to append
// a new goal at the end of the list.
func (r *GoalRepository) CountForTopic(ctx context.Context, userID, topicID string) (int, error) {
	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return 0, err
	}
	tid, err := mongo.IDFromHex(topicID)
	if err != nil {
		return 0, err
	}
	n, err := r.repo.CountBy(ctx, bson.M{"userId": uid, "topicId": tid})
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// CountsByTopics returns the progress of each requested topic in a single
// aggregation: one round-trip regardless of how many topics are listed. Topics
// with no goals are absent from the map (callers treat that as 0%).
func (r *GoalRepository) CountsByTopics(ctx context.Context, userID string, topicIDs []string) (map[string]domain.Progress, error) {
	result := make(map[string]domain.Progress, len(topicIDs))
	if len(topicIDs) == 0 {
		return result, nil
	}

	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return nil, err
	}
	oids := make([]primitive.ObjectID, 0, len(topicIDs))
	for _, id := range topicIDs {
		tid, err := mongo.IDFromHex(id)
		if err != nil {
			return nil, err
		}
		oids = append(oids, tid)
	}

	pipeline := []bson.D{
		{{Key: "$match", Value: bson.M{"userId": uid, "topicId": bson.M{"$in": oids}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$topicId"},
			{Key: "total", Value: bson.M{"$sum": 1}},
			{Key: "completed", Value: bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$status", string(domain.GoalStatusDone)}}, 1, 0,
			}}}},
		}}},
	}

	var rows []struct {
		ID        primitive.ObjectID `bson:"_id"`
		Total     int                `bson:"total"`
		Completed int                `bson:"completed"`
	}
	if err := r.repo.Aggregate(ctx, pipeline, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[mongo.HexFromID(row.ID)] = domain.ComputeProgress(row.Total, row.Completed)
	}
	return result, nil
}

// goalUpdateFields turns a domain GoalUpdate into the bson $set fields. Done maps
// to the status transition and its completedAt timestamp.
func goalUpdateFields(u domain.GoalUpdate) bson.M {
	fields := bson.M{}
	if u.Title != nil {
		fields["title"] = strings.TrimSpace(*u.Title)
	}
	if u.Notes != nil {
		fields["notes"] = strings.TrimSpace(*u.Notes)
	}
	if u.Order != nil {
		fields["order"] = *u.Order
	}
	if u.Done != nil {
		if *u.Done {
			fields["status"] = string(domain.GoalStatusDone)
			fields["completedAt"] = time.Now().UTC()
		} else {
			fields["status"] = string(domain.GoalStatusTodo)
			fields["completedAt"] = nil
		}
	}
	return fields
}
