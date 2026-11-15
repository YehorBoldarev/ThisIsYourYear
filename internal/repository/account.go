package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"

	"kibou/thisisyouryear/internal/platform/mongo"
)

// Whole-account operations for the operator's `admin delete-user` command.
// They are still scoped by user: each one touches only documents whose userId
// is the given account.

// userScope is the filter selecting every document a user owns.
func userScope(userID string) (bson.M, error) {
	uid, err := mongo.IDFromHex(userID)
	if err != nil {
		return nil, err
	}
	return bson.M{"userId": uid}, nil
}

// CountForUser returns how many goals the user has across all topics.
func (r *GoalRepository) CountForUser(ctx context.Context, userID string) (int64, error) {
	filter, err := userScope(userID)
	if err != nil {
		return 0, err
	}
	return r.repo.CountBy(ctx, filter)
}

// DeleteAllForUser deletes every goal the user has, returning how many.
func (r *GoalRepository) DeleteAllForUser(ctx context.Context, userID string) (int64, error) {
	filter, err := userScope(userID)
	if err != nil {
		return 0, err
	}
	return r.repo.DeleteManyBy(ctx, filter)
}

// CountForUser returns how many topics the user has, in every year.
func (r *TopicRepository) CountForUser(ctx context.Context, userID string) (int64, error) {
	filter, err := userScope(userID)
	if err != nil {
		return 0, err
	}
	return r.repo.CountBy(ctx, filter)
}

// DeleteAllForUser deletes every topic the user has, returning how many.
func (r *TopicRepository) DeleteAllForUser(ctx context.Context, userID string) (int64, error) {
	filter, err := userScope(userID)
	if err != nil {
		return 0, err
	}
	return r.repo.DeleteManyBy(ctx, filter)
}

// CountForUser returns how many refresh sessions (active or revoked) the user has.
func (r *SessionRepository) CountForUser(ctx context.Context, userID string) (int64, error) {
	filter, err := userScope(userID)
	if err != nil {
		return 0, err
	}
	return r.repo.CountBy(ctx, filter)
}

// DeleteAllForUser deletes every refresh session the user has, returning how many.
func (r *SessionRepository) DeleteAllForUser(ctx context.Context, userID string) (int64, error) {
	filter, err := userScope(userID)
	if err != nil {
		return 0, err
	}
	return r.repo.DeleteManyBy(ctx, filter)
}

// Delete removes the user document itself, or returns domain.ErrNotFound.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	return r.repo.Delete(ctx, id)
}
