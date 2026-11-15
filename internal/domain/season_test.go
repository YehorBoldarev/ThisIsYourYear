package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	require.NoError(t, err)
	return loc
}

func TestSeasonAt_PreparationStage(t *testing.T) {
	kyiv := mustLoc(t, "Europe/Kyiv")
	now := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)

	s := SeasonAt(now, kyiv, 2027)
	require.Equal(t, 2027, s.Year)
	require.True(t, s.Preparing)
	require.Equal(t, PhasePreparation, s.Phase())
	require.True(t, s.CountdownTarget().Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, kyiv)),
		"counts down to local midnight on 1 January 2027")
	require.Equal(t, "2026-12-31T22:00:00Z", s.CountdownTarget().UTC().Format(time.RFC3339))
	require.Equal(t, int64(s.CountdownTarget().Sub(now).Seconds()), s.SecondsRemaining(),
		"measured from the instant the season was resolved for")
}

func TestSeasonAt_NormalYear(t *testing.T) {
	now := time.Date(2027, 6, 15, 12, 0, 0, 0, time.UTC)

	for name, prep := range map[string]int{"disabled": 0, "past": 2027, "long past": 2020} {
		t.Run(name, func(t *testing.T) {
			s := SeasonAt(now, time.UTC, prep)
			require.Equal(t, 2027, s.Year)
			require.False(t, s.Preparing)
			require.Equal(t, PhaseYear, s.Phase())
			require.True(t, s.CountdownTarget().Equal(time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)),
				"counts down to the next New Year")
		})
	}
}

// The stage must end at each user's own local midnight, not the server's.
func TestSeasonAt_MidnightBoundaryPerTimezone(t *testing.T) {
	for _, name := range []string{"UTC", "Europe/Kyiv", "Europe/Warsaw", "America/New_York", "Pacific/Kiritimati"} {
		t.Run(name, func(t *testing.T) {
			loc := mustLoc(t, name)
			lastSecond := time.Date(2026, 12, 31, 23, 59, 59, 0, loc)
			midnight := time.Date(2027, 1, 1, 0, 0, 0, 0, loc)

			before := SeasonAt(lastSecond, loc, 2027)
			require.True(t, before.Preparing, "still preparing one second before local midnight")
			require.Equal(t, 2027, before.Year)

			after := SeasonAt(midnight, loc, 2027)
			require.False(t, after.Preparing, "the year starts at local midnight")
			require.Equal(t, 2027, after.Year)
			require.True(t, after.CountdownTarget().Equal(time.Date(2028, 1, 1, 0, 0, 0, 0, loc)))
		})
	}
}

// The same instant can be 2026 in one timezone and 2027 in another.
func TestSeasonAt_SameInstantDifferentTimezones(t *testing.T) {
	instant := time.Date(2026, 12, 31, 22, 30, 0, 0, time.UTC) // 00:30 in Kyiv, 23:30 in Warsaw
	require.False(t, SeasonAt(instant, mustLoc(t, "Europe/Kyiv"), 2027).Preparing)
	require.True(t, SeasonAt(instant, mustLoc(t, "Europe/Warsaw"), 2027).Preparing)
}

func TestSeasonAt_NilLocationIsUTC(t *testing.T) {
	now := time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC)
	require.Equal(t, SeasonAt(now, time.UTC, 2027), SeasonAt(now, nil, 2027))
}
