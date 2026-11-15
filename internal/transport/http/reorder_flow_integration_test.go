//go:build integration

package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// Milestone reordering (plan D3, Part A) over real HTTP and MongoDB.

func goalTitles(goals []GoalResponse) []string {
	out := make([]string, len(goals))
	for i, g := range goals {
		out[i] = g.Title
	}
	return out
}

func (a *testApp) listGoals(t *testing.T, token, topicID string) []GoalResponse {
	t.Helper()
	resp := a.doJSON(t, http.MethodGet, "/api/v1/topics/"+topicID+"/goals", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var goals []GoalResponse
	decodeBody(t, resp, &goals)
	return goals
}

func TestReorderFlow(t *testing.T) {
	app := startApp(t)
	token := app.registerAndLogin(t, "arranger@example.com")
	topic := app.createTopic(t, token, "Run a half-marathon")
	a := app.createGoal(t, token, topic, "A")
	b := app.createGoal(t, token, topic, "B")
	c := app.createGoal(t, token, topic, "C")

	resp := app.doJSON(t, http.MethodPut, "/api/v1/topics/"+topic+"/goals/order",
		ReorderGoalsRequest{GoalIDs: []string{c, a, b}}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var reordered []GoalResponse
	decodeBody(t, resp, &reordered)
	require.Equal(t, []string{"C", "A", "B"}, goalTitles(reordered))
	require.Equal(t, []string{"C", "A", "B"}, goalTitles(app.listGoals(t, token, topic)), "persisted")

	// A list that doesn't match the topic (e.g. a stale screen) is rejected.
	body := decodeError(t, app.doJSON(t, http.MethodPut, "/api/v1/topics/"+topic+"/goals/order",
		ReorderGoalsRequest{GoalIDs: []string{a, b}}, token), http.StatusUnprocessableEntity)
	require.Contains(t, body.Error.Fields, "goalIds")

	// A new milestone after a deletion goes last, with no position clash.
	resp = app.doJSON(t, http.MethodDelete, "/api/v1/goals/"+a, nil, token)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
	app.createGoal(t, token, topic, "D")
	require.Equal(t, []string{"C", "B", "D"}, goalTitles(app.listGoals(t, token, topic)))

	// Someone else can't reorder this topic.
	other := app.registerAndLogin(t, "intruder@example.com")
	resp = app.doJSON(t, http.MethodPut, "/api/v1/topics/"+topic+"/goals/order",
		ReorderGoalsRequest{GoalIDs: []string{b, c}}, other)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}
