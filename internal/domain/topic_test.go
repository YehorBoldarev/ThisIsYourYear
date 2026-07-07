package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTopic_Valid(t *testing.T) {
	topic, err := NewTopic("user-1", "  Get fit  ", "  run more  ", 2026, "#aabbcc", "🏃")
	require.NoError(t, err)
	require.Equal(t, "Get fit", topic.Title)
	require.Equal(t, "run more", topic.Description)
	require.Equal(t, 2026, topic.Year)
	require.Equal(t, "#aabbcc", topic.Color)
	require.False(t, topic.IsArchived())
	require.Empty(t, topic.ID)
}

func TestNewTopic_DefaultsYearToCurrent(t *testing.T) {
	topic, err := NewTopic("user-1", "Read books", "", 0, "", "")
	require.NoError(t, err)
	require.Equal(t, CurrentYear(), topic.Year)
}

func TestNewTopic_Invariants(t *testing.T) {
	cases := map[string]struct {
		userID, title, desc string
		year                int
		color, icon         string
	}{
		"no user":        {"", "T", "", 2026, "", ""},
		"empty title":    {"u", "   ", "", 2026, "", ""},
		"title too long": {"u", strings.Repeat("x", TopicTitleMaxLen+1), "", 2026, "", ""},
		"desc too long":  {"u", "T", strings.Repeat("x", TopicDescriptionMaxLen+1), 2026, "", ""},
		"bad color":      {"u", "T", "", 2026, "red", ""},
		"year too low":   {"u", "T", "", 1999, "", ""},
		"year too high":  {"u", "T", "", 2101, "", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewTopic(c.userID, c.title, c.desc, c.year, c.color, c.icon)
			require.ErrorIs(t, err, ErrValidation)
		})
	}
}

func TestTopicUpdate_IsEmpty(t *testing.T) {
	require.True(t, TopicUpdate{}.IsEmpty())
	title := "x"
	require.False(t, TopicUpdate{Title: &title}.IsEmpty())
}

func TestTopicUpdate_Validate(t *testing.T) {
	empty := ""
	badYear := 1000
	good := "New title"

	require.ErrorIs(t, TopicUpdate{Title: &empty}.Validate(), ErrValidation)
	require.ErrorIs(t, TopicUpdate{Year: &badYear}.Validate(), ErrValidation)
	require.NoError(t, TopicUpdate{Title: &good}.Validate())
	require.NoError(t, TopicUpdate{}.Validate())
}
