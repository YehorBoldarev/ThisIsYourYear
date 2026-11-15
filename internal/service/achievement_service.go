package service

import (
	"context"

	"kibou/thisisyouryear/internal/domain"
)

// AchievementTopics lists a user's topics for a year (TopicRepository).
type AchievementTopics interface {
	ListForUser(ctx context.Context, userID string, year *int, includeArchived bool) ([]domain.Topic, error)
}

// AchievementGoals lists a user's finished, starred goals under some topics,
// newest first (GoalRepository).
type AchievementGoals interface {
	ListAchievements(ctx context.Context, userID string, topicIDs []string) ([]domain.Goal, error)
}

// Achievement is one "win": a finished, starred goal with the topic it belongs to.
type Achievement struct {
	Goal  domain.Goal
	Topic domain.Topic
}

// AchievementService builds the Wins list in two queries (the year's topics,
// then their wins) instead of one goals request per topic.
type AchievementService struct {
	topics AchievementTopics
	goals  AchievementGoals
}

// NewAchievementService wires an AchievementService.
func NewAchievementService(topics AchievementTopics, goals AchievementGoals) *AchievementService {
	return &AchievementService{topics: topics, goals: goals}
}

// List returns the caller's wins across their non-archived topics of year,
// newest completion first.
func (s *AchievementService) List(ctx context.Context, userID string, year int) ([]Achievement, error) {
	if err := domain.ValidateYear(year); err != nil {
		return nil, err
	}
	topics, err := s.topics.ListForUser(ctx, userID, &year, false)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]domain.Topic, len(topics))
	ids := make([]string, 0, len(topics))
	for _, t := range topics {
		byID[t.ID] = t
		ids = append(ids, t.ID)
	}
	goals, err := s.goals.ListAchievements(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Achievement, 0, len(goals))
	for _, g := range goals {
		if t, ok := byID[g.TopicID]; ok {
			out = append(out, Achievement{Goal: g, Topic: t})
		}
	}
	return out, nil
}
