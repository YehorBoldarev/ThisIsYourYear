package service

import (
	"context"
	"fmt"

	"kibou/thisisyouryear/internal/domain"
)

// TopicRepository is the persistence contract TopicService needs. Every method
// is scoped by userID so ownership is enforced at the data layer.
type TopicRepository interface {
	Create(ctx context.Context, t *domain.Topic) error
	ByIDForUser(ctx context.Context, id, userID string) (*domain.Topic, error)
	ListForUser(ctx context.Context, userID string, year *int, includeArchived bool) ([]domain.Topic, error)
	UpdateForUser(ctx context.Context, id, userID string, upd domain.TopicUpdate) error
	DeleteForUser(ctx context.Context, id, userID string) error
}

// TopicGoals is the goal-side collaborator a topic needs: progress counts for
// its goals and cascade deletion. GoalRepository satisfies it.
type TopicGoals interface {
	CountsByTopics(ctx context.Context, userID string, topicIDs []string) (map[string]domain.Progress, error)
	DeleteByTopic(ctx context.Context, userID, topicID string) error
}

// TopicView is a topic together with its computed progress — the unit the
// transport layer renders.
type TopicView struct {
	Topic    *domain.Topic
	Progress domain.Progress
}

// TopicService implements the topic use-cases. All operations are keyed by the
// authenticated user's id.
type TopicService struct {
	topics TopicRepository
	goals  TopicGoals
}

// NewTopicService wires a TopicService. The goals port supplies live progress
// and cascade-on-delete.
func NewTopicService(topics TopicRepository, goals TopicGoals) *TopicService {
	return &TopicService{topics: topics, goals: goals}
}

// Create validates and persists a new topic for the user.
func (s *TopicService) Create(ctx context.Context, userID, title, description string, year int, color, icon string) (*TopicView, error) {
	topic, err := domain.NewTopic(userID, title, description, year, color, icon)
	if err != nil {
		return nil, err
	}
	if err := s.topics.Create(ctx, topic); err != nil {
		return nil, err
	}
	// A brand-new topic has no goals yet, so its progress is 0% without a query.
	return &TopicView{Topic: topic, Progress: domain.ComputeProgress(0, 0)}, nil
}

// List returns the user's topics for a year (defaulting to the current year when
// year is nil), excluding archived topics unless requested.
func (s *TopicService) List(ctx context.Context, userID string, year *int, includeArchived bool) ([]TopicView, error) {
	scopeYear := domain.CurrentYear()
	if year != nil {
		if err := domain.ValidateYear(*year); err != nil {
			return nil, err
		}
		scopeYear = *year
	}

	topics, err := s.topics.ListForUser(ctx, userID, &scopeYear, includeArchived)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(topics))
	for i := range topics {
		ids[i] = topics[i].ID
	}
	progress, err := s.goals.CountsByTopics(ctx, userID, ids)
	if err != nil {
		return nil, err
	}

	views := make([]TopicView, 0, len(topics))
	for i := range topics {
		views = append(views, TopicView{Topic: &topics[i], Progress: progress[topics[i].ID]})
	}
	return views, nil
}

// Get returns one of the user's topics with its live progress, or
// domain.ErrNotFound.
func (s *TopicService) Get(ctx context.Context, userID, id string) (*TopicView, error) {
	topic, err := s.topics.ByIDForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	progress, err := s.goals.CountsByTopics(ctx, userID, []string{topic.ID})
	if err != nil {
		return nil, err
	}
	return &TopicView{Topic: topic, Progress: progress[topic.ID]}, nil
}

// Update applies a partial update to the user's topic and returns the fresh view.
func (s *TopicService) Update(ctx context.Context, userID, id string, upd domain.TopicUpdate) (*TopicView, error) {
	if upd.IsEmpty() {
		return nil, fmt.Errorf("%w: no fields to update", domain.ErrValidation)
	}
	if err := upd.Validate(); err != nil {
		return nil, err
	}
	if err := s.topics.UpdateForUser(ctx, id, userID, upd); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

// Delete removes the user's topic and cascades to its goals. The topic delete is
// ownership-checked first (a non-owned topic yields ErrNotFound and nothing is
// cascaded).
func (s *TopicService) Delete(ctx context.Context, userID, id string) error {
	if err := s.topics.DeleteForUser(ctx, id, userID); err != nil {
		return err
	}
	return s.goals.DeleteByTopic(ctx, userID, id)
}
