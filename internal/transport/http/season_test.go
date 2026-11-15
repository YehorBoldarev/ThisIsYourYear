package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func seasonRequest(tz string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	if tz != "" {
		r.Header.Set(ClientTimezoneHeader, tz)
	}
	return r
}

func TestSeasonResolver_FollowsClientTimezone(t *testing.T) {
	// 22:30 UTC on New Year's Eve: already 2027 in Kyiv, still 2026 in Warsaw.
	now := time.Date(2026, 12, 31, 22, 30, 0, 0, time.UTC)
	res := NewSeasonResolver(SeasonConfig{PreparationYear: 2027, Now: func() time.Time { return now }})

	kyiv := res.Resolve(seasonRequest("Europe/Kyiv"))
	require.False(t, kyiv.Preparing)
	require.Equal(t, 2027, kyiv.Year)

	warsaw := res.Resolve(seasonRequest("Europe/Warsaw"))
	require.True(t, warsaw.Preparing)
	require.Equal(t, "2026-12-31T23:00:00Z", warsaw.CountdownTarget().UTC().Format(time.RFC3339))
}

func TestSeasonResolver_FallsBackToDefaultLocation(t *testing.T) {
	now := time.Date(2026, 12, 31, 22, 30, 0, 0, time.UTC)
	kyiv, err := time.LoadLocation("Europe/Kyiv")
	require.NoError(t, err)
	res := NewSeasonResolver(SeasonConfig{
		PreparationYear: 2027, DefaultLocation: kyiv, Now: func() time.Time { return now },
	})

	for name, tz := range map[string]string{
		"missing":   "",
		"unknown":   "Mars/Olympus_Mons",
		"traversal": "../../etc/passwd",
		"oversized": strings.Repeat("A", 200),
	} {
		t.Run(name, func(t *testing.T) {
			s := res.Resolve(seasonRequest(tz))
			require.Equal(t, kyiv.String(), s.StartsAt.Location().String(), "falls back to APP_TIMEZONE")
			require.False(t, s.Preparing, "already 2027 in Kyiv")
		})
	}
}

func TestSeasonResolver_ZeroConfigIsUTCWithoutPreparation(t *testing.T) {
	res := NewSeasonResolver(SeasonConfig{})
	s := res.Resolve(seasonRequest(""))
	require.False(t, s.Preparing)
	require.Equal(t, time.Now().UTC().Year(), s.Year)
	require.Equal(t, time.UTC, s.StartsAt.Location())
}

func TestSeasonResolver_CachesOnlyValidZones(t *testing.T) {
	res := NewSeasonResolver(SeasonConfig{})
	res.Resolve(seasonRequest("Europe/Kyiv"))
	res.Resolve(seasonRequest("Not/A_Zone"))

	cached := 0
	res.locations.Range(func(_, _ any) bool { cached++; return true })
	require.Equal(t, 1, cached)
}
