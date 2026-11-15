package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

// fakeAchievementGoals returns canned wins and records the topic ids it was asked about.
type fakeAchievementGoals struct {
	wins  []domain.Goal
	asked []string
	calls int
}

func (f *fakeAchievementGoals) ListAchievements(_ context.Context, _ string, topicIDs []string) ([]domain.Goal, error) {
	f.calls++
	f.asked = topicIDs
	in := map[string]bool{}
	for _, id := range topicIDs {
		in[id] = true
	}
	out := []domain.Goal{}
	for _, g := range f.wins {
		if in[g.TopicID] {
			out = append(out, g)
		}
	}
	return out, nil
}

func TestAchievementService_ListJoinsTopicsInOneGoalQuery(t *testing.T) {
	topics := newMemTopicRepo()
	ctx := context.Background()
	mk := func(title string, year int) domain.Topic {
		tp, err := domain.NewTopic("u", title, "", year, "#3ec5a0", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, topics.Create(ctx, tp))
		return *tp
	}
	run := mk("Run", 2027)
	read := mk("Read", 2027)
	old := mk("Old", 2026)

	done := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	goals := &fakeAchievementGoals{wins: []domain.Goal{
		{ID: "g2", TopicID: read.ID, Title: "10 books", CompletedAt: &done},
		{ID: "g1", TopicID: run.ID, Title: "First 10k", CompletedAt: &done},
		{ID: "g0", TopicID: old.ID, Title: "Last year's win", CompletedAt: &done},
	}}
	svc := NewAchievementService(topics, goals)

	wins, err := svc.List(ctx, "u", 2027)
	require.NoError(t, err)
	require.Equal(t, 1, goals.calls, "one goal query for all topics")
	require.ElementsMatch(t, []string{run.ID, read.ID}, goals.asked, "only that year's topics")
	require.Len(t, wins, 2)
	require.Equal(t, "10 books", wins[0].Goal.Title, "keeps the repository's newest-first order")
	require.Equal(t, "Read", wins[0].Topic.Title)
	require.Equal(t, "Run", wins[1].Topic.Title)
}

func TestAchievementService_EmptyYearAndInvalidYear(t *testing.T) {
	svc := NewAchievementService(newMemTopicRepo(), &fakeAchievementGoals{})
	wins, err := svc.List(context.Background(), "u", 2027)
	require.NoError(t, err)
	require.Empty(t, wins)

	_, err = svc.List(context.Background(), "u", 1999)
	require.True(t, domain.IsValidation(err))
}
