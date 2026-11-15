package service

import (
	"context"
	"fmt"
	"time"

	"kibou/thisisyouryear/internal/domain"
)

// GoalRepository is the persistence contract GoalService needs. Every method is
// scoped by userID so ownership is enforced at the data layer.
type GoalRepository interface {
	Create(ctx context.Context, g *domain.Goal) error
	ByIDForUser(ctx context.Context, id, userID string) (*domain.Goal, error)
	ListForTopic(ctx context.Context, userID, topicID string) ([]domain.Goal, error)
	UpdateForUser(ctx context.Context, id, userID string, upd domain.GoalUpdate) error
	DeleteForUser(ctx context.Context, id, userID string) error
	NextOrder(ctx context.Context, userID, topicID string) (int, error)
	SetOrder(ctx context.Context, userID, topicID string, goalIDs []string) error
}

// TopicOwnershipReader lets the goal service confirm a topic belongs to the
// caller before operating on its goals. TopicRepository satisfies it.
type TopicOwnershipReader interface {
	ByIDForUser(ctx context.Context, id, userID string) (*domain.Topic, error)
}

// GoalService implements the goal use-cases, keyed by the authenticated user.
type GoalService struct {
	goals  GoalRepository
	topics TopicOwnershipReader
}

// NewGoalService wires a GoalService.
func NewGoalService(goals GoalRepository, topics TopicOwnershipReader) *GoalService {
	return &GoalService{goals: goals, topics: topics}
}

// CreateGoalInput carries the fields for creating a milestone under a topic.
type CreateGoalInput struct {
	Title         string
	Notes         string
	DueDate       *time.Time
	IsAchievement bool
}

// Create adds a goal to one of the caller's topics, appended at the end of the
// list. A topic the caller does not own yields domain.ErrNotFound.
func (s *GoalService) Create(ctx context.Context, userID, topicID string, in CreateGoalInput) (*domain.Goal, error) {
	if err := s.assertOwnsTopic(ctx, userID, topicID); err != nil {
		return nil, err
	}
	goal, err := domain.NewGoal(userID, topicID, in.Title, in.Notes, in.IsAchievement, in.DueDate)
	if err != nil {
		return nil, err
	}
	// After the current last goal; a count would collide once deletions leave gaps.
	next, err := s.goals.NextOrder(ctx, userID, topicID)
	if err != nil {
		return nil, err
	}
	goal.Order = next
	if err := s.goals.Create(ctx, goal); err != nil {
		return nil, err
	}
	return goal, nil
}

// ListForTopic returns the caller's goals under a topic, ordered.
func (s *GoalService) ListForTopic(ctx context.Context, userID, topicID string) ([]domain.Goal, error) {
	if err := s.assertOwnsTopic(ctx, userID, topicID); err != nil {
		return nil, err
	}
	return s.goals.ListForTopic(ctx, userID, topicID)
}

// Update applies a partial update (edit / reorder / mark done or undone) to the
// caller's goal and returns the fresh goal.
func (s *GoalService) Update(ctx context.Context, userID, goalID string, upd domain.GoalUpdate) (*domain.Goal, error) {
	if upd.IsEmpty() {
		return nil, fmt.Errorf("%w: no fields to update", domain.ErrValidation)
	}
	if err := upd.Validate(); err != nil {
		return nil, err
	}
	if err := s.goals.UpdateForUser(ctx, goalID, userID, upd); err != nil {
		return nil, err
	}
	return s.goals.ByIDForUser(ctx, goalID, userID)
}

// Reorder sets the order of a topic's goals: goalIDs must list every goal under
// the topic exactly once (so a stale client can't reorder a list that has
// changed meanwhile), else a *domain.FieldError on "goalIds". Returns the goals
// in their new order. A topic the caller does not own yields domain.ErrNotFound.
func (s *GoalService) Reorder(ctx context.Context, userID, topicID string, goalIDs []string) ([]domain.Goal, error) {
	if err := s.assertOwnsTopic(ctx, userID, topicID); err != nil {
		return nil, err
	}
	current, err := s.goals.ListForTopic(ctx, userID, topicID)
	if err != nil {
		return nil, err
	}
	if !sameSet(current, goalIDs) {
		return nil, &domain.FieldError{Field: "goalIds", Message: "must list every milestone of this goal exactly once"}
	}
	if err := s.goals.SetOrder(ctx, userID, topicID, goalIDs); err != nil {
		return nil, err
	}
	return s.goals.ListForTopic(ctx, userID, topicID)
}

// sameSet reports whether ids names exactly the given goals, each once.
func sameSet(goals []domain.Goal, ids []string) bool {
	if len(goals) != len(ids) {
		return false
	}
	want := make(map[string]bool, len(goals))
	for _, g := range goals {
		want[g.ID] = true
	}
	for _, id := range ids {
		if !want[id] {
			return false // unknown, or a duplicate (already consumed)
		}
		delete(want, id)
	}
	return true
}

// Delete removes the caller's goal.
func (s *GoalService) Delete(ctx context.Context, userID, goalID string) error {
	return s.goals.DeleteForUser(ctx, goalID, userID)
}

// assertOwnsTopic returns domain.ErrNotFound if the topic is missing or not the
// caller's, so goal operations never touch another user's topic.
func (s *GoalService) assertOwnsTopic(ctx context.Context, userID, topicID string) error {
	_, err := s.topics.ByIDForUser(ctx, topicID, userID)
	return err
}
