//go:build integration

package http

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Preparation stage (plan D1, Part A) over real HTTP and MongoDB: while the
// configured year lies ahead, the dashboard counts down to its start in the
// client's timezone, and plans made now belong to that year.

// getWithTimezone performs an authenticated GET carrying X-Client-Timezone.
func (a *testApp) getWithTimezone(t *testing.T, path, token, tz string, dst any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, a.server.URL+path, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(ClientTimezoneHeader, tz)
	resp, err := a.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(dst))
}

func TestPreparationStage_DashboardAndDefaultYear(t *testing.T) {
	now := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	app := startAppWith(t, appOptions{Season: SeasonConfig{
		PreparationYear: 2027,
		Now:             func() time.Time { return now },
	}})
	token := app.registerAndLogin(t, "planner@example.com")

	var dash DashboardResponse
	app.getWithTimezone(t, "/api/v1/dashboard", token, "Europe/Kyiv", &dash)
	require.Equal(t, "preparation", dash.Phase)
	require.Equal(t, 2027, dash.Year)
	require.Equal(t, "2026-12-31T22:00:00Z", dash.Countdown.Target.UTC().Format(time.RFC3339),
		"local midnight in Kyiv")
	require.Equal(t, int64(dash.Countdown.Target.Sub(now).Seconds()), dash.Countdown.SecondsRemaining)
	require.Contains(t, dash.Message, "2027")

	// Another timezone, another midnight.
	app.getWithTimezone(t, "/api/v1/dashboard", token, "America/New_York", &dash)
	require.Equal(t, "2027-01-01T05:00:00Z", dash.Countdown.Target.UTC().Format(time.RFC3339))

	// A goal created now (no explicit year) belongs to 2027 and is what the
	// default topic list and the dashboard show.
	app.createTopic(t, token, "Run a half-marathon")
	var topics []TopicResponse
	app.getWithTimezone(t, "/api/v1/topics", token, "Europe/Kyiv", &topics)
	require.Len(t, topics, 1)
	require.Equal(t, 2027, topics[0].Year)

	app.getWithTimezone(t, "/api/v1/dashboard", token, "Europe/Kyiv", &dash)
	require.Equal(t, 1, dash.Topics.Total)
}

func TestPreparationStage_EndsAtLocalMidnight(t *testing.T) {
	// 22:30 UTC on New Year's Eve: 00:30 in Kyiv (2027 has begun), 23:30 in Warsaw.
	now := time.Date(2026, 12, 31, 22, 30, 0, 0, time.UTC)
	app := startAppWith(t, appOptions{Season: SeasonConfig{
		PreparationYear: 2027,
		Now:             func() time.Time { return now },
	}})
	token := app.registerAndLogin(t, "newyear@example.com")

	var kyiv, warsaw DashboardResponse
	app.getWithTimezone(t, "/api/v1/dashboard", token, "Europe/Kyiv", &kyiv)
	app.getWithTimezone(t, "/api/v1/dashboard", token, "Europe/Warsaw", &warsaw)

	require.Equal(t, "year", kyiv.Phase, "the year has started in Kyiv")
	require.Equal(t, 2027, kyiv.Year)
	require.Equal(t, "2027-12-31T22:00:00Z", kyiv.Countdown.Target.UTC().Format(time.RFC3339),
		"now counting down to 2028 (Kyiv midnight)")

	require.Equal(t, "preparation", warsaw.Phase, "still New Year's Eve in Warsaw")
	require.Equal(t, int64(30*60), warsaw.Countdown.SecondsRemaining)
}
