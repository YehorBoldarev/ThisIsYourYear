// Package repository holds the concrete MongoDB implementations of the
// persistence interfaces the service layer depends on. Each repository wraps the
// generic mongo.Repository over a private document model and maps that model to
// and from the pure domain entity, so the domain never carries storage tags and
// the service layer never sees Mongo types.
package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/platform/mongo"
)

const usersCollection = "users"

// userDoc is the persistence model for a user. It embeds mongo.Document to gain
// the identity and audit-timestamp fields the generic repository manages.
type userDoc struct {
	mongo.Document `bson:",inline"`
	Email          string `bson:"email"`
	PasswordHash   string `bson:"passwordHash"`
	DisplayName    string `bson:"displayName"`
}

func (d *userDoc) toDomain() *domain.User {
	return &domain.User{
		ID:           mongo.HexFromID(d.ID),
		Email:        d.Email,
		PasswordHash: d.PasswordHash,
		DisplayName:  d.DisplayName,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

// UserRepository persists users and enforces the unique-email constraint.
type UserRepository struct {
	repo *mongo.Repository[userDoc, *userDoc]
}

// NewUserRepository builds the repository and registers the users indexes with
// the client so EnsureIndexes creates them on startup.
func NewUserRepository(client *mongo.Client, clock mongo.Clock) *UserRepository {
	client.RegisterIndexes(usersCollection, mongo.IndexSpec{
		Keys:   bson.D{{Key: "email", Value: 1}},
		Unique: true,
		Name:   "uniq_email",
	})
	return &UserRepository{
		repo: mongo.NewRepository[userDoc, *userDoc](client.DB(), usersCollection, clock),
	}
}

// Create persists a new user. A duplicate email surfaces as domain.ErrConflict.
// On success the user's ID and timestamps are populated from the stored document.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	doc := &userDoc{
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		DisplayName:  u.DisplayName,
	}
	if err := r.repo.Insert(ctx, doc); err != nil {
		return err
	}
	u.ID = mongo.HexFromID(doc.ID)
	u.CreatedAt = doc.CreatedAt
	u.UpdatedAt = doc.UpdatedAt
	return nil
}

// ByEmail looks up a user by (normalized) email, or domain.ErrNotFound.
func (r *UserRepository) ByEmail(ctx context.Context, email string) (*domain.User, error) {
	doc, err := r.repo.FindOne(ctx, bson.M{"email": domain.NormalizeEmail(email)})
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// ByID looks up a user by hex id, or domain.ErrNotFound.
func (r *UserRepository) ByID(ctx context.Context, id string) (*domain.User, error) {
	doc, err := r.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// SetPasswordHash replaces a user's password hash (admin password reset), or
// returns domain.ErrNotFound for an unknown id.
func (r *UserRepository) SetPasswordHash(ctx context.Context, id, passwordHash string) error {
	return r.repo.Update(ctx, id, bson.M{"passwordHash": passwordHash})
}

// List returns every user, oldest first. It is deliberately not scoped by user:
// it serves only the operator's `admin list-users` command, never an HTTP route.
func (r *UserRepository) List(ctx context.Context) ([]*domain.User, error) {
	docs, err := r.repo.List(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	users := make([]*domain.User, 0, len(docs))
	for i := range docs {
		users = append(users, docs[i].toDomain())
	}
	return users, nil
}
