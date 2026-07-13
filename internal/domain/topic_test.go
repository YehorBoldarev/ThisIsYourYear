package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewTopic_Valid(t *testing.T) {
	target := time.Date(2026, time.December, 31, 15, 4, 5, 0, time.UTC)
	topic, err := NewTopic("user-1", "  Get fit  ", "  run more  ", 2026, "#aabbcc", "🏃", "  Health  ", &target)
	require.NoError(t, err)
	require.Equal(t, "Get fit", topic.Title)
	require.Equal(t, "run more", topic.Description)
	require.Equal(t, 2026, topic.Year)
	require.Equal(t, "#aabbcc", topic.Color)
	require.Equal(t, "Health", topic.Category, "category is trimmed")
	require.NotNil(t, topic.TargetDate)
	require.Equal(t, time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC), *topic.TargetDate,
		"target date is normalized to UTC midnight")
	require.False(t, topic.IsArchived())
	require.Empty(t, topic.ID)
}

func TestNewTopic_DefaultsYearToCurrent(t *testing.T) {
	topic, err := NewTopic("user-1", "Read books", "", 0, "", "", "", nil)
	require.NoError(t, err)
	require.Equal(t, CurrentYear(), topic.Year)
	require.Nil(t, topic.TargetDate, "no target date stays nil")
}

func TestNewTopic_Invariants(t *testing.T) {
	cases := map[string]struct {
		userID, title, desc string
		year                int
		color, icon, cat    string
	}{
		"no user":           {"", "T", "", 2026, "", "", ""},
		"empty title":       {"u", "   ", "", 2026, "", "", ""},
		"title too long":    {"u", strings.Repeat("x", TopicTitleMaxLen+1), "", 2026, "", "", ""},
		"desc too long":     {"u", "T", strings.Repeat("x", TopicDescriptionMaxLen+1), 2026, "", "", ""},
		"category too long": {"u", "T", "", 2026, "", "", strings.Repeat("x", TopicCategoryMaxLen+1)},
		"bad color":         {"u", "T", "", 2026, "red", "", ""},
		"year too low":      {"u", "T", "", 1999, "", "", ""},
		"year too high":     {"u", "T", "", 2101, "", "", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewTopic(c.userID, c.title, c.desc, c.year, c.color, c.icon, c.cat, nil)
			require.ErrorIs(t, err, ErrValidation)
		})
	}
}

func TestTopicUpdate_IsEmpty(t *testing.T) {
	require.True(t, TopicUpdate{}.IsEmpty())
	title := "x"
	require.False(t, TopicUpdate{Title: &title}.IsEmpty())
	cat := "Health"
	require.False(t, TopicUpdate{Category: &cat}.IsEmpty())
	// A present target-date update (even a clear) is a change.
	require.False(t, TopicUpdate{TargetDate: ClearField[time.Time]()}.IsEmpty())
}

func TestTopicUpdate_Validate(t *testing.T) {
	empty := ""
	badYear := 1000
	good := "New title"
	longCat := strings.Repeat("x", TopicCategoryMaxLen+1)

	require.ErrorIs(t, TopicUpdate{Title: &empty}.Validate(), ErrValidation)
	require.ErrorIs(t, TopicUpdate{Year: &badYear}.Validate(), ErrValidation)
	require.ErrorIs(t, TopicUpdate{Category: &longCat}.Validate(), ErrValidation)
	require.NoError(t, TopicUpdate{Title: &good}.Validate())
	require.NoError(t, TopicUpdate{}.Validate())
}

func TestTopicUpdate_NormalizedTargetDate(t *testing.T) {
	// Setting a value normalizes to UTC midnight; clearing and leaving pass through.
	instant := time.Date(2026, time.March, 4, 9, 30, 0, 0, time.UTC)
	set := TopicUpdate{TargetDate: SetField(instant)}.NormalizedTargetDate()
	require.True(t, set.Present)
	require.NotNil(t, set.Value)
	require.Equal(t, time.Date(2026, time.March, 4, 0, 0, 0, 0, time.UTC), *set.Value)

	clear := TopicUpdate{TargetDate: ClearField[time.Time]()}.NormalizedTargetDate()
	require.True(t, clear.Present)
	require.Nil(t, clear.Value)

	require.False(t, TopicUpdate{}.NormalizedTargetDate().Present)
}
