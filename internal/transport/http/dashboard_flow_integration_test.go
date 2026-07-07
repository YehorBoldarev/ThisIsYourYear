//go:build integration

package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

func TestDashboardFlow_EmptyThenAggregates(t *testing.T) {
	app := startApp(t)
	token := app.registerAndLogin(t, "dash@example.com")

	// Empty state: no topics yet.
	empty := getDashboard(t, app, token)
	require.Equal(t, domain.CurrentYear(), empty.Year)
	require.Equal(t, 0, empty.Topics.Total)
	require.Equal(t, 0, empty.Topics.Completed)
	require.Equal(t, 0, empty.Progress.Total)
	require.Equal(t,
		domain.DashboardMessage(domain.ComputeProgress(0, 0), 0),
		empty.Message,
		"no topics => empty-state message")
	require.Positive(t, empty.Countdown.SecondsRemaining)
	require.Equal(t, domain.CurrentYear()+1, empty.Countdown.Target.Year(),
		"countdown targets the next New Year")
	require.True(t, empty.Countdown.Target.After(time.Now()))

	// Two topics: one with 1 of 2 goals done, one fully complete (2 of 2).
	partial := app.createTopic(t, token, "Partial")
	complete := app.createTopic(t, token, "Complete")

	partialGoals := []string{
		app.createGoal(t, token, partial, "p1"),
		app.createGoal(t, token, partial, "p2"),
	}
	completeGoals := []string{
		app.createGoal(t, token, complete, "c1"),
		app.createGoal(t, token, complete, "c2"),
	}
	app.completeGoal(t, token, partialGoals[0])
	for _, id := range completeGoals {
		app.completeGoal(t, token, id)
	}

	// Dashboard now aggregates: 4 goals total, 3 done, 1 topic complete.
	view := getDashboard(t, app, token)
	require.Equal(t, 2, view.Topics.Total)
	require.Equal(t, 1, view.Topics.Completed, "only the fully-done topic counts as complete")
	require.Equal(t, 4, view.Progress.Total)
	require.Equal(t, 3, view.Progress.Completed)
	require.InDelta(t, 0.75, view.Progress.Ratio, 1e-9)
	require.Equal(t, domain.Motivate(domain.ComputeProgress(4, 3)), view.Message,
		"message tracks the overall progress")
}

func TestDashboardFlow_Unauthenticated(t *testing.T) {
	app := startApp(t)
	resp := app.doJSON(t, http.MethodGet, "/api/v1/dashboard", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestDashboardFlow_IsPerUser(t *testing.T) {
	app := startApp(t)
	alice := app.registerAndLogin(t, "alice-dash@example.com")
	bob := app.registerAndLogin(t, "bob-dash@example.com")

	topic := app.createTopic(t, alice, "Alice's topic")
	app.completeGoal(t, alice, app.createGoal(t, alice, topic, "g1"))

	// Bob's dashboard sees none of Alice's data.
	bobView := getDashboard(t, app, bob)
	require.Equal(t, 0, bobView.Topics.Total)
	require.Equal(t, 0, bobView.Progress.Total)
}

// getDashboard fetches and decodes the dashboard for a user.
func getDashboard(t *testing.T, app *testApp, token string) DashboardResponse {
	t.Helper()
	resp := app.doJSON(t, http.MethodGet, "/api/v1/dashboard", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var view DashboardResponse
	decodeBody(t, resp, &view)
	return view
}

// createGoal adds a goal to a topic and returns its id.
func (a *testApp) createGoal(t *testing.T, token, topicID, title string) string {
	t.Helper()
	resp := a.doJSON(t, http.MethodPost, "/api/v1/topics/"+topicID+"/goals", CreateGoalRequest{Title: title}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var g GoalResponse
	decodeBody(t, resp, &g)
	return g.ID
}

// completeGoal marks a goal done.
func (a *testApp) completeGoal(t *testing.T, token, goalID string) {
	t.Helper()
	done := true
	resp := a.doJSON(t, http.MethodPatch, "/api/v1/goals/"+goalID, UpdateGoalRequest{Done: &done}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}
