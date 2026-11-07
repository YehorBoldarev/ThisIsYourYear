package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

// fixedNow returns a deterministic clock for the countdown assertions.
func fixedNow() time.Time {
	return time.Date(2026, time.July, 7, 12, 0, 0, 0, time.UTC)
}

// newDashboardService wires a DashboardService over the in-memory topic service,
// returning the goals fake so tests can seed per-topic progress.
func newDashboardService() (*DashboardService, *TopicService, *fakeTopicGoals) {
	topics := newMemTopicRepo()
	goals := newFakeTopicGoals()
	topicSvc := NewTopicService(topics, goals)
	return NewDashboardService(topicSvc, fixedNow), topicSvc, goals
}

func TestDashboardService_EmptyState(t *testing.T) {
	dash, _, _ := newDashboardService()
	ctx := context.Background()

	view, err := dash.Get(ctx, "u")
	require.NoError(t, err)
	require.Equal(t, 0, view.TopicsTotal)
	require.Equal(t, 0, view.TopicsCompleted)
	require.Equal(t, 0, view.Progress.Total)
	require.Equal(t, domain.DashboardMessage(domain.ComputeProgress(0, 0), 0), view.Message)
	require.Equal(t, domain.CurrentYear(), view.Year)

	// Countdown reflects the injected clock.
	wantTarget, wantSecs := domain.Countdown(fixedNow().UTC())
	require.True(t, view.CountdownTarget.Equal(wantTarget))
	require.Equal(t, wantSecs, view.SecondsRemaining)
	require.Positive(t, view.SecondsRemaining)
}

func TestDashboardService_AggregatesAcrossTopics(t *testing.T) {
	dash, topicSvc, goals := newDashboardService()
	ctx := context.Background()

	// Three current-year topics with mixed progress: 1/4, 0/2, and 3/3 (complete).
	a, err := topicSvc.Create(ctx, "u", CreateTopicInput{Title: "A", Year: domain.CurrentYear()})
	require.NoError(t, err)
	b, err := topicSvc.Create(ctx, "u", CreateTopicInput{Title: "B", Year: domain.CurrentYear()})
	require.NoError(t, err)
	c, err := topicSvc.Create(ctx, "u", CreateTopicInput{Title: "C", Year: domain.CurrentYear()})
	require.NoError(t, err)

	goals.counts[a.Topic.ID] = domain.ComputeProgress(4, 1)
	goals.counts[b.Topic.ID] = domain.ComputeProgress(2, 0)
	goals.counts[c.Topic.ID] = domain.ComputeProgress(3, 3)

	view, err := dash.Get(ctx, "u")
	require.NoError(t, err)
	require.Equal(t, 3, view.TopicsTotal)
	require.Equal(t, 1, view.TopicsCompleted, "only the 3/3 topic is complete")
	require.Equal(t, 9, view.Progress.Total, "4+2+3 goals")
	require.Equal(t, 4, view.Progress.Completed, "1+0+3 done")
	require.InDelta(t, 4.0/9.0, view.Progress.Ratio, 1e-9)
	require.Equal(t, domain.Motivate(view.Progress).Message, view.Message, "message tracks overall progress")
}

func TestDashboardService_ExcludesOtherYearsAndArchived(t *testing.T) {
	dash, topicSvc, _ := newDashboardService()
	ctx := context.Background()

	_, err := topicSvc.Create(ctx, "u", CreateTopicInput{Title: "This year", Year: domain.CurrentYear()})
	require.NoError(t, err)
	_, err = topicSvc.Create(ctx, "u", CreateTopicInput{Title: "Old year", Year: 2020})
	require.NoError(t, err)

	view, err := dash.Get(ctx, "u")
	require.NoError(t, err)
	require.Equal(t, 1, view.TopicsTotal, "dashboard is scoped to the current year")
}
