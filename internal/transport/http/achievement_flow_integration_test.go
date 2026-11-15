//go:build integration

package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Wins endpoint (plan D3, Part B) over real HTTP and MongoDB.

func (a *testApp) createStarredGoal(t *testing.T, token, topicID, title string) string {
	t.Helper()
	resp := a.doJSON(t, http.MethodPost, "/api/v1/topics/"+topicID+"/goals",
		CreateGoalRequest{Title: title, IsAchievement: true}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var g GoalResponse
	decodeBody(t, resp, &g)
	return g.ID
}

func getAchievements(t *testing.T, app *testApp, path, token string) []AchievementResponse {
	t.Helper()
	resp := app.doJSON(t, http.MethodGet, path, nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var wins []AchievementResponse
	decodeBody(t, resp, &wins)
	return wins
}

func TestAchievementFlow_OnlyFinishedStarredGoalsNewestFirst(t *testing.T) {
	app := startApp(t)
	token := app.registerAndLogin(t, "wins@example.com")

	empty := getAchievements(t, app, "/api/v1/achievements", token)
	require.Empty(t, empty)
	require.NotNil(t, empty, "an empty list, not null")

	run := app.createTopic(t, token, "Run a half-marathon")
	read := app.createTopic(t, token, "Read 24 books")

	first := app.createStarredGoal(t, token, run, "First 10k")
	app.completeGoal(t, token, first)
	time.Sleep(10 * time.Millisecond) // distinct completedAt
	second := app.createStarredGoal(t, token, read, "Finished book 12")
	app.completeGoal(t, token, second)

	app.createStarredGoal(t, token, run, "Register for the race") // starred, not done
	plain := app.createGoal(t, token, read, "Buy a reading lamp")
	app.completeGoal(t, token, plain) // done, not starred

	wins := getAchievements(t, app, "/api/v1/achievements", token)
	require.Len(t, wins, 2)
	require.Equal(t, "Finished book 12", wins[0].Goal.Title, "newest first")
	require.Equal(t, "Read 24 books", wins[0].Topic.Title)
	require.Equal(t, "First 10k", wins[1].Goal.Title)
	require.Equal(t, run, wins[1].Topic.ID)

	// Another year has none; a bad year is a 422.
	require.Empty(t, getAchievements(t, app, "/api/v1/achievements?year=2020", token))
	resp := app.doJSON(t, http.MethodGet, "/api/v1/achievements?year=1999", nil, token)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	resp.Body.Close()

	// Wins are private.
	other := app.registerAndLogin(t, "someone-else@example.com")
	require.Empty(t, getAchievements(t, app, "/api/v1/achievements", other))

	resp = app.doJSON(t, http.MethodGet, "/api/v1/achievements", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}
