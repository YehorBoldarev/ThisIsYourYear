package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/platform/mongo"
)

const sessionsCollection = "sessions"

// sessionDoc is the persistence model for a refresh session. userId is stored as
// an ObjectID reference to the owning user rather than a hex string, so it joins
// and indexes like a native reference.
type sessionDoc struct {
	mongo.Document   `bson:",inline"`
	UserID           primitive.ObjectID `bson:"userId"`
	RefreshTokenHash string             `bson:"refreshTokenHash"`
	RememberMe       bool               `bson:"rememberMe"`
	UserAgent        string             `bson:"userAgent"`
	IP               string             `bson:"ip"`
	ExpiresAt        time.Time          `bson:"expiresAt"`
	RevokedAt        *time.Time         `bson:"revokedAt,omitempty"`
}

func (d *sessionDoc) toDomain() *domain.Session {
	return &domain.Session{
		ID:               mongo.HexFromID(d.ID),
		UserID:           mongo.HexFromID(d.UserID),
		RefreshTokenHash: d.RefreshTokenHash,
		RememberMe:       d.RememberMe,
		UserAgent:        d.UserAgent,
		IP:               d.IP,
		ExpiresAt:        d.ExpiresAt,
		RevokedAt:        d.RevokedAt,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

// SessionRepository persists refresh sessions and manages their lifecycle.
type SessionRepository struct {
	repo *mongo.Repository[sessionDoc, *sessionDoc]
}

// NewSessionRepository builds the repository and registers the session indexes:
// a TTL index that reaps documents at expiresAt, a unique index on the token
// hash (fast lookup + no hash collisions), and an index on userId for bulk
// revocation.
func NewSessionRepository(client *mongo.Client, clock mongo.Clock) *SessionRepository {
	reapAtExpiry := time.Duration(0) // expireAfterSeconds: 0 => remove exactly at expiresAt
	client.RegisterIndexes(sessionsCollection,
		mongo.IndexSpec{
			Keys: bson.D{{Key: "expiresAt", Value: 1}},
			TTL:  &reapAtExpiry,
			Name: "ttl_expiresAt",
		},
		mongo.IndexSpec{
			Keys:   bson.D{{Key: "refreshTokenHash", Value: 1}},
			Unique: true,
			Name:   "uniq_refreshTokenHash",
		},
		mongo.IndexSpec{
			Keys: bson.D{{Key: "userId", Value: 1}},
			Name: "idx_userId",
		},
	)
	return &SessionRepository{
		repo: mongo.NewRepository[sessionDoc, *sessionDoc](client.DB(), sessionsCollection, clock),
	}
}

// Create persists a new session. On success the session's ID and timestamps are
// populated from the stored document.
func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	uid, err := mongo.IDFromHex(s.UserID)
	if err != nil {
		return err
	}
	doc := &sessionDoc{
		UserID:           uid,
		RefreshTokenHash: s.RefreshTokenHash,
		RememberMe:       s.RememberMe,
		UserAgent:        s.UserAgent,
		IP:               s.IP,
		ExpiresAt:        s.ExpiresAt,
		RevokedAt:        s.RevokedAt,
	}
	if err := r.repo.Insert(ctx, doc); err != nil {
		return err
	}
	s.ID = mongo.HexFromID(doc.ID)
	s.CreatedAt = doc.CreatedAt
	s.UpdatedAt = doc.UpdatedAt
	return nil
}

// ByTokenHash looks up a session by its stored refresh-token hash, or
// domain.ErrNotFound.
func (r *SessionRepository) ByTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	doc, err := r.repo.FindOne(ctx, bson.M{"refreshTokenHash": hash})
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// Revoke marks a single session revoked at the given time.
func (r *SessionRepository) Revoke(ctx context.Context, id string, at time.Time) error {
	return r.repo.Update(ctx, id, bson.M{"revokedAt": at})
}

// RevokeAllForUser revokes every currently-active session belonging to a user.
// It is used to invalidate a whole session lineage when refresh-token reuse is
// detected. Revoking zero sessions is not an error.
func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID string, at time.Time) error {
	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return err
	}
	_, err = r.repo.UpdateMany(ctx,
		bson.M{"userId": uid, "revokedAt": nil},
		bson.M{"revokedAt": at},
	)
	return err
}
