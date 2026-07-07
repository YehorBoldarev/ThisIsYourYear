package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

// memTopicRepo is an in-memory TopicRepository that enforces ownership exactly
// like the real one, so service logic can be tested without a database.
type memTopicRepo struct {
	topics map[string]*domain.Topic // keyed by id
	seq    int
}

func newMemTopicRepo() *memTopicRepo { return &memTopicRepo{topics: map[string]*domain.Topic{}} }

func (m *memTopicRepo) Create(_ context.Context, t *domain.Topic) error {
	m.seq++
	t.ID = fmt.Sprintf("topic-%d", m.seq)
	stored := *t
	m.topics[t.ID] = &stored
	return nil
}

func (m *memTopicRepo) ByIDForUser(_ context.Context, id, userID string) (*domain.Topic, error) {
	t, ok := m.topics[id]
	if !ok || t.UserID != userID {
		return nil, domain.ErrNotFound
	}
	clone := *t
	return &clone, nil
}

func (m *memTopicRepo) ListForUser(_ context.Context, userID string, year *int, includeArchived bool) ([]domain.Topic, error) {
	var out []domain.Topic
	for _, t := range m.topics {
		if t.UserID != userID {
			continue
		}
		if year != nil && t.Year != *year {
			continue
		}
		if !includeArchived && t.IsArchived() {
			continue
		}
		out = append(out, *t)
	}
	return out, nil
}

func (m *memTopicRepo) UpdateForUser(_ context.Context, id, userID string, upd domain.TopicUpdate) error {
	t, ok := m.topics[id]
	if !ok || t.UserID != userID {
		return domain.ErrNotFound
	}
	if upd.Title != nil {
		t.Title = *upd.Title
	}
	if upd.Year != nil {
		t.Year = *upd.Year
	}
	return nil
}

func (m *memTopicRepo) DeleteForUser(_ context.Context, id, userID string) error {
	t, ok := m.topics[id]
	if !ok || t.UserID != userID {
		return domain.ErrNotFound
	}
	delete(m.topics, id)
	return nil
}

// fakeTopicGoals is an in-memory TopicGoals port for testing topic progress and
// cascade without a real goals repository.
type fakeTopicGoals struct {
	counts        map[string]domain.Progress // topicID -> progress
	deletedTopics []string
}

func newFakeTopicGoals() *fakeTopicGoals {
	return &fakeTopicGoals{counts: map[string]domain.Progress{}}
}

func (f *fakeTopicGoals) CountsByTopics(_ context.Context, _ string, ids []string) (map[string]domain.Progress, error) {
	out := make(map[string]domain.Progress, len(ids))
	for _, id := range ids {
		if p, ok := f.counts[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

func (f *fakeTopicGoals) DeleteByTopic(_ context.Context, _, topicID string) error {
	f.deletedTopics = append(f.deletedTopics, topicID)
	return nil
}

// newTopicService builds a TopicService over fresh in-memory collaborators,
// returning the goals fake so tests can seed progress counts and inspect cascade.
func newTopicService() (*TopicService, *fakeTopicGoals) {
	goals := newFakeTopicGoals()
	return NewTopicService(newMemTopicRepo(), goals), goals
}

func TestTopicService_CreateAndGet(t *testing.T) {
	svc, _ := newTopicService()
	ctx := context.Background()

	view, err := svc.Create(ctx, "user-1", "Get fit", "", 2026, "", "")
	require.NoError(t, err)
	require.NotEmpty(t, view.Topic.ID)
	require.Equal(t, 0, view.Progress.Total, "no goals yet => 0 progress")
	require.InDelta(t, 0.0, view.Progress.Ratio, 1e-9)

	got, err := svc.Get(ctx, "user-1", view.Topic.ID)
	require.NoError(t, err)
	require.Equal(t, "Get fit", got.Topic.Title)
}

func TestTopicService_OwnershipIsolation(t *testing.T) {
	svc, _ := newTopicService()
	ctx := context.Background()
	view, err := svc.Create(ctx, "owner", "Mine", "", 2026, "", "")
	require.NoError(t, err)

	// Another user cannot read, update, or delete it.
	_, err = svc.Get(ctx, "intruder", view.Topic.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	newTitle := "Hijacked"
	_, err = svc.Update(ctx, "intruder", view.Topic.ID, domain.TopicUpdate{Title: &newTitle})
	require.ErrorIs(t, err, domain.ErrNotFound)

	err = svc.Delete(ctx, "intruder", view.Topic.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestTopicService_List_DefaultsToCurrentYear(t *testing.T) {
	svc, _ := newTopicService()
	ctx := context.Background()
	_, err := svc.Create(ctx, "u", "This year", "", domain.CurrentYear(), "", "")
	require.NoError(t, err)
	_, err = svc.Create(ctx, "u", "Old year", "", 2020, "", "")
	require.NoError(t, err)

	current, err := svc.List(ctx, "u", nil, false)
	require.NoError(t, err)
	require.Len(t, current, 1, "nil year defaults to the current year")
	require.Equal(t, "This year", current[0].Topic.Title)

	y := 2020
	old, err := svc.List(ctx, "u", &y, false)
	require.NoError(t, err)
	require.Len(t, old, 1)
	require.Equal(t, "Old year", old[0].Topic.Title)
}

func TestTopicService_Update_EmptyIsValidationError(t *testing.T) {
	svc, _ := newTopicService()
	ctx := context.Background()
	view, err := svc.Create(ctx, "u", "T", "", 2026, "", "")
	require.NoError(t, err)

	_, err = svc.Update(ctx, "u", view.Topic.ID, domain.TopicUpdate{})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestTopicService_Update_RejectsInvalidFields(t *testing.T) {
	svc, _ := newTopicService()
	ctx := context.Background()
	view, err := svc.Create(ctx, "u", "T", "", 2026, "", "")
	require.NoError(t, err)

	empty := ""
	_, err = svc.Update(ctx, "u", view.Topic.ID, domain.TopicUpdate{Title: &empty})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestTopicService_LiveProgress(t *testing.T) {
	svc, goals := newTopicService()
	ctx := context.Background()
	view, err := svc.Create(ctx, "u", "Get fit", "", 2026, "", "")
	require.NoError(t, err)

	// Simulate 1 of 4 goals done for this topic.
	goals.counts[view.Topic.ID] = domain.ComputeProgress(4, 1)

	got, err := svc.Get(ctx, "u", view.Topic.ID)
	require.NoError(t, err)
	require.Equal(t, 4, got.Progress.Total)
	require.Equal(t, 1, got.Progress.Completed)
	require.InDelta(t, 0.25, got.Progress.Ratio, 1e-9)

	list, err := svc.List(ctx, "u", nil, false)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.InDelta(t, 0.25, list[0].Progress.Ratio, 1e-9)
}

func TestTopicService_Delete_CascadesGoals(t *testing.T) {
	svc, goals := newTopicService()
	ctx := context.Background()
	view, err := svc.Create(ctx, "u", "Doomed", "", 2026, "", "")
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, "u", view.Topic.ID))
	require.Equal(t, []string{view.Topic.ID}, goals.deletedTopics, "topic delete cascades to its goals")
}

func TestTopicService_Delete_NotOwned_DoesNotCascade(t *testing.T) {
	svc, goals := newTopicService()
	ctx := context.Background()
	view, err := svc.Create(ctx, "owner", "Mine", "", 2026, "", "")
	require.NoError(t, err)

	err = svc.Delete(ctx, "intruder", view.Topic.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Empty(t, goals.deletedTopics, "a rejected delete must not cascade")
}
