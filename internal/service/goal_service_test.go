package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

// memGoalRepo is an in-memory GoalRepository enforcing ownership like the real one.
type memGoalRepo struct {
	goals map[string]*domain.Goal // keyed by id
	seq   int
}

func newMemGoalRepo() *memGoalRepo { return &memGoalRepo{goals: map[string]*domain.Goal{}} }

func (m *memGoalRepo) Create(_ context.Context, g *domain.Goal) error {
	m.seq++
	g.ID = fmt.Sprintf("goal-%d", m.seq)
	stored := *g
	m.goals[g.ID] = &stored
	return nil
}

func (m *memGoalRepo) ByIDForUser(_ context.Context, id, userID string) (*domain.Goal, error) {
	g, ok := m.goals[id]
	if !ok || g.UserID != userID {
		return nil, domain.ErrNotFound
	}
	clone := *g
	return &clone, nil
}

func (m *memGoalRepo) ListForTopic(_ context.Context, userID, topicID string) ([]domain.Goal, error) {
	var out []domain.Goal
	for _, g := range m.goals {
		if g.UserID == userID && g.TopicID == topicID {
			out = append(out, *g)
		}
	}
	return out, nil
}

func (m *memGoalRepo) UpdateForUser(_ context.Context, id, userID string, upd domain.GoalUpdate) error {
	g, ok := m.goals[id]
	if !ok || g.UserID != userID {
		return domain.ErrNotFound
	}
	if upd.Title != nil {
		g.Title = *upd.Title
	}
	if upd.Done != nil {
		if *upd.Done {
			g.Status = domain.GoalStatusDone
		} else {
			g.Status = domain.GoalStatusTodo
		}
	}
	if upd.Order != nil {
		g.Order = *upd.Order
	}
	return nil
}

func (m *memGoalRepo) DeleteForUser(_ context.Context, id, userID string) error {
	g, ok := m.goals[id]
	if !ok || g.UserID != userID {
		return domain.ErrNotFound
	}
	delete(m.goals, id)
	return nil
}

func (m *memGoalRepo) CountForTopic(_ context.Context, userID, topicID string) (int, error) {
	n := 0
	for _, g := range m.goals {
		if g.UserID == userID && g.TopicID == topicID {
			n++
		}
	}
	return n, nil
}

// stubTopicOwner reports ownership for a fixed (user, topic) pair.
type stubTopicOwner struct {
	owner   string
	topicID string
}

func (s stubTopicOwner) ByIDForUser(_ context.Context, id, userID string) (*domain.Topic, error) {
	if id == s.topicID && userID == s.owner {
		return &domain.Topic{ID: id, UserID: userID}, nil
	}
	return nil, domain.ErrNotFound
}

// testTopicID is the single topic the goal-service tests operate under.
const testTopicID = "topic-1"

func newGoalService(owner string) *GoalService {
	return NewGoalService(newMemGoalRepo(), stubTopicOwner{owner: owner, topicID: testTopicID})
}

func TestGoalService_CreateAppendsOrder(t *testing.T) {
	svc := newGoalService("u")
	ctx := context.Background()

	g0, err := svc.Create(ctx, "u", "topic-1", CreateGoalInput{Title: "First"})
	require.NoError(t, err)
	require.Equal(t, 0, g0.Order)
	require.Equal(t, domain.GoalStatusTodo, g0.Status)

	g1, err := svc.Create(ctx, "u", "topic-1", CreateGoalInput{Title: "Second"})
	require.NoError(t, err)
	require.Equal(t, 1, g1.Order, "new goals are appended at the end")
}

func TestGoalService_Create_TopicNotOwned(t *testing.T) {
	svc := newGoalService("owner")
	ctx := context.Background()

	// Wrong user for the topic.
	_, err := svc.Create(ctx, "intruder", "topic-1", CreateGoalInput{Title: "X"})
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Unknown topic.
	_, err = svc.Create(ctx, "owner", "topic-other", CreateGoalInput{Title: "X"})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGoalService_MarkDoneAndUndone(t *testing.T) {
	svc := newGoalService("u")
	ctx := context.Background()
	g, err := svc.Create(ctx, "u", "topic-1", CreateGoalInput{Title: "Run"})
	require.NoError(t, err)

	done := true
	updated, err := svc.Update(ctx, "u", g.ID, domain.GoalUpdate{Done: &done})
	require.NoError(t, err)
	require.True(t, updated.IsAchieved())

	undone := false
	reverted, err := svc.Update(ctx, "u", g.ID, domain.GoalUpdate{Done: &undone})
	require.NoError(t, err)
	require.Equal(t, domain.GoalStatusTodo, reverted.Status)
}

func TestGoalService_Update_EmptyIsValidationError(t *testing.T) {
	svc := newGoalService("u")
	ctx := context.Background()
	g, err := svc.Create(ctx, "u", "topic-1", CreateGoalInput{Title: "Run"})
	require.NoError(t, err)

	_, err = svc.Update(ctx, "u", g.ID, domain.GoalUpdate{})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestGoalService_OwnershipIsolation(t *testing.T) {
	svc := newGoalService("owner")
	ctx := context.Background()
	g, err := svc.Create(ctx, "owner", "topic-1", CreateGoalInput{Title: "Private"})
	require.NoError(t, err)

	newTitle := "Hijacked"
	_, err = svc.Update(ctx, "intruder", g.ID, domain.GoalUpdate{Title: &newTitle})
	require.ErrorIs(t, err, domain.ErrNotFound)

	err = svc.Delete(ctx, "intruder", g.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}
