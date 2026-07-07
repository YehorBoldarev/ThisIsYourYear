//go:build integration

package http

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoalFlow_CRUDAndProgress(t *testing.T) {
	app := startApp(t)
	token := app.registerAndLogin(t, "owner@example.com")
	topicID := app.createTopic(t, token, "Get fit")

	// Create three goals; they are appended in order.
	var ids []string
	for i, title := range []string{"Run 5k", "Do 20 pushups", "Sleep 8h"} {
		resp := app.doJSON(t, http.MethodPost, "/api/v1/topics/"+topicID+"/goals",
			CreateGoalRequest{Title: title}, token)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var g GoalResponse
		decodeBody(t, resp, &g)
		require.Equal(t, i, g.Order, "goals are appended in order")
		require.Equal(t, "todo", g.Status)
		ids = append(ids, g.ID)
	}

	// List returns them ordered.
	resp := app.doJSON(t, http.MethodGet, "/api/v1/topics/"+topicID+"/goals", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var goals []GoalResponse
	decodeBody(t, resp, &goals)
	require.Len(t, goals, 3)
	require.Equal(t, "Run 5k", goals[0].Title)

	// Topic progress starts at 0 of 3.
	require.InDelta(t, 0.0, topicProgress(t, app, token, topicID).Ratio, 1e-9)

	// Mark the first goal done -> it becomes an achievement, progress rises.
	done := true
	resp = app.doJSON(t, http.MethodPatch, "/api/v1/goals/"+ids[0], UpdateGoalRequest{Done: &done}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var completed GoalResponse
	decodeBody(t, resp, &completed)
	require.Equal(t, "done", completed.Status)
	require.NotNil(t, completed.CompletedAt)

	prog := topicProgress(t, app, token, topicID)
	require.Equal(t, 3, prog.Total)
	require.Equal(t, 1, prog.Completed)
	require.InDelta(t, 1.0/3.0, prog.Ratio, 1e-9)

	// Undo it -> back to todo, progress drops.
	undone := false
	resp = app.doJSON(t, http.MethodPatch, "/api/v1/goals/"+ids[0], UpdateGoalRequest{Done: &undone}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	require.InDelta(t, 0.0, topicProgress(t, app, token, topicID).Ratio, 1e-9)

	// Reorder: move the first goal to the end by giving it a higher order.
	// (Ordering is an explicit settable int; the client assigns the value.)
	last := 5
	resp = app.doJSON(t, http.MethodPatch, "/api/v1/goals/"+ids[0], UpdateGoalRequest{Order: &last}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics/"+topicID+"/goals", nil, token)
	decodeBody(t, resp, &goals)
	require.Equal(t, "Run 5k", goals[len(goals)-1].Title, "reordered goal moved to the end")

	// Delete a goal.
	resp = app.doJSON(t, http.MethodDelete, "/api/v1/goals/"+ids[1], nil, token)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics/"+topicID+"/goals", nil, token)
	decodeBody(t, resp, &goals)
	require.Len(t, goals, 2)
}

func TestGoalFlow_TopicDeleteCascades(t *testing.T) {
	app := startApp(t)
	token := app.registerAndLogin(t, "cascade@example.com")
	topicID := app.createTopic(t, token, "Doomed")

	resp := app.doJSON(t, http.MethodPost, "/api/v1/topics/"+topicID+"/goals", CreateGoalRequest{Title: "g1"}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Delete the topic; its goals go with it, and the topic (and its goals
	// endpoint) is gone.
	resp = app.doJSON(t, http.MethodDelete, "/api/v1/topics/"+topicID, nil, token)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics/"+topicID+"/goals", nil, token)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "goals under a deleted topic are unreachable")
	resp.Body.Close()
}

func TestGoalFlow_OwnershipIsolation(t *testing.T) {
	app := startApp(t)
	ownerToken := app.registerAndLogin(t, "alice@example.com")
	intruderToken := app.registerAndLogin(t, "mallory@example.com")

	topicID := app.createTopic(t, ownerToken, "Alice's plan")
	resp := app.doJSON(t, http.MethodPost, "/api/v1/topics/"+topicID+"/goals", CreateGoalRequest{Title: "secret"}, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var goal GoalResponse
	decodeBody(t, resp, &goal)

	// Mallory can neither create under nor list Alice's topic.
	resp = app.doJSON(t, http.MethodPost, "/api/v1/topics/"+topicID+"/goals", CreateGoalRequest{Title: "intrude"}, intruderToken)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics/"+topicID+"/goals", nil, intruderToken)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	// ...nor patch/delete Alice's goal by id.
	hijack := "hijacked"
	resp = app.doJSON(t, http.MethodPatch, "/api/v1/goals/"+goal.ID, UpdateGoalRequest{Title: &hijack}, intruderToken)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
	resp = app.doJSON(t, http.MethodDelete, "/api/v1/goals/"+goal.ID, nil, intruderToken)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestGoalFlow_Validation(t *testing.T) {
	app := startApp(t)
	token := app.registerAndLogin(t, "val@example.com")
	topicID := app.createTopic(t, token, "T")

	// Missing title -> 422.
	resp := app.doJSON(t, http.MethodPost, "/api/v1/topics/"+topicID+"/goals", CreateGoalRequest{Notes: "no title"}, token)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	resp.Body.Close()

	// Goal under a non-existent topic -> 404.
	resp = app.doJSON(t, http.MethodPost, "/api/v1/topics/"+fakeObjectID+"/goals", CreateGoalRequest{Title: "x"}, token)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

// fakeObjectID is a syntactically valid but non-existent ObjectID hex.
const fakeObjectID = "0123456789abcdef01234567"

// topicProgress fetches a topic and returns its progress block.
func topicProgress(t *testing.T, app *testApp, token, topicID string) ProgressResponse {
	t.Helper()
	resp := app.doJSON(t, http.MethodGet, "/api/v1/topics/"+topicID, nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode, fmt.Sprintf("get topic %s", topicID))
	var topic TopicResponse
	decodeBody(t, resp, &topic)
	return topic.Progress
}
