package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewGoal_Valid(t *testing.T) {
	g, err := NewGoal("user-1", "topic-1", "  Run 5km  ", "  couch to 5k  ")
	require.NoError(t, err)
	require.Equal(t, "Run 5km", g.Title)
	require.Equal(t, "couch to 5k", g.Notes)
	require.Equal(t, GoalStatusTodo, g.Status)
	require.False(t, g.IsAchieved())
	require.Nil(t, g.CompletedAt)
	require.Empty(t, g.ID)
}

func TestNewGoal_Invariants(t *testing.T) {
	cases := map[string]struct{ userID, topicID, title, notes string }{
		"no user":        {"", "t", "Title", ""},
		"no topic":       {"u", "", "Title", ""},
		"empty title":    {"u", "t", "   ", ""},
		"title too long": {"u", "t", strings.Repeat("x", GoalTitleMaxLen+1), ""},
		"notes too long": {"u", "t", "Title", strings.Repeat("x", GoalNotesMaxLen+1)},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewGoal(c.userID, c.topicID, c.title, c.notes)
			require.ErrorIs(t, err, ErrValidation)
		})
	}
}

func TestGoalUpdate_IsEmpty(t *testing.T) {
	require.True(t, GoalUpdate{}.IsEmpty())
	done := true
	require.False(t, GoalUpdate{Done: &done}.IsEmpty())
}

func TestGoalUpdate_Validate(t *testing.T) {
	empty := ""
	neg := -1
	good := "New title"

	require.ErrorIs(t, GoalUpdate{Title: &empty}.Validate(), ErrValidation)
	require.ErrorIs(t, GoalUpdate{Order: &neg}.Validate(), ErrValidation)
	require.NoError(t, GoalUpdate{Title: &good}.Validate())
	require.NoError(t, GoalUpdate{}.Validate())
}
