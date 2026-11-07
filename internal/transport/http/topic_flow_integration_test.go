//go:build integration

package http

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

func TestTopicFlow_CRUD(t *testing.T) {
	app := startApp(t)
	token := app.registerAndLogin(t, "owner@example.com")

	// Create.
	resp := app.doJSON(t, http.MethodPost, "/api/v1/topics", CreateTopicRequest{
		Title: "Get fit", Description: "move more", Color: "#aabbcc", Icon: "🏃",
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created TopicResponse
	decodeBody(t, resp, &created)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "Get fit", created.Title)
	require.Equal(t, domain.CurrentYear(), created.Year, "year defaults to current")
	require.Equal(t, 0, created.Progress.Total, "no goals yet")
	require.Equal(t, motivationResponse(domain.Motivate(domain.ComputeProgress(0, 0))), created.Motivation,
		"a topic with no goals carries the no-goals message")
	require.False(t, created.Archived)

	// Get.
	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var fetched TopicResponse
	decodeBody(t, resp, &fetched)
	require.Equal(t, created.ID, fetched.ID)

	// List (current year) includes it.
	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var list []TopicResponse
	decodeBody(t, resp, &list)
	require.Len(t, list, 1)

	// Patch.
	newTitle := "Get very fit"
	resp = app.doJSON(t, http.MethodPatch, "/api/v1/topics/"+created.ID, UpdateTopicRequest{
		Title: &newTitle,
	}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var patched TopicResponse
	decodeBody(t, resp, &patched)
	require.Equal(t, "Get very fit", patched.Title)

	// Archive via patch, then it drops out of the default list.
	archived := true
	resp = app.doJSON(t, http.MethodPatch, "/api/v1/topics/"+created.ID, UpdateTopicRequest{
		Archived: &archived,
	}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics", nil, token)
	decodeBody(t, resp, &list)
	require.Empty(t, list, "archived topics are hidden by default")

	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics?includeArchived=true", nil, token)
	decodeBody(t, resp, &list)
	require.Len(t, list, 1, "archived topics appear when requested")

	// Delete.
	resp = app.doJSON(t, http.MethodDelete, "/api/v1/topics/"+created.ID, nil, token)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics/"+created.ID, nil, token)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestTopicFlow_OwnershipIsolation(t *testing.T) {
	app := startApp(t)
	ownerToken := app.registerAndLogin(t, "alice@example.com")
	intruderToken := app.registerAndLogin(t, "mallory@example.com")

	// Alice creates a topic.
	resp := app.doJSON(t, http.MethodPost, "/api/v1/topics", CreateTopicRequest{Title: "Alice's plan"}, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var topic TopicResponse
	decodeBody(t, resp, &topic)

	// Mallory cannot see it in her list...
	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics", nil, intruderToken)
	var malloryList []TopicResponse
	decodeBody(t, resp, &malloryList)
	require.Empty(t, malloryList)

	// ...nor GET/PATCH/DELETE it by id (all 404, existence not leaked).
	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics/"+topic.ID, nil, intruderToken)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	hijack := "Hijacked"
	resp = app.doJSON(t, http.MethodPatch, "/api/v1/topics/"+topic.ID, UpdateTopicRequest{Title: &hijack}, intruderToken)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	resp = app.doJSON(t, http.MethodDelete, "/api/v1/topics/"+topic.ID, nil, intruderToken)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	// Alice's topic is untouched.
	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics/"+topic.ID, nil, ownerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var stillThere TopicResponse
	decodeBody(t, resp, &stillThere)
	require.Equal(t, "Alice's plan", stillThere.Title)
}

// TestTopicFlow_DesignFields round-trips the design-driven fields (category and
// the DD-MM-YYYY target date) through create, read, patch-change, and clear.
func TestTopicFlow_DesignFields(t *testing.T) {
	app := startApp(t)
	token := app.registerAndLogin(t, "fields-topic@example.com")

	// Create with a category and a target date.
	resp := app.doJSON(t, http.MethodPost, "/api/v1/topics", CreateTopicRequest{
		Title: "Run a Half-Marathon", Category: "Health & fitness", TargetDate: "31-12-2026",
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created TopicResponse
	decodeBody(t, resp, &created)
	require.Equal(t, "Health & fitness", created.Category)
	require.Equal(t, "31-12-2026", created.TargetDate)

	// Change the target date via PATCH.
	resp = app.doJSON(t, http.MethodPatch, "/api/v1/topics/"+created.ID, UpdateTopicRequest{
		TargetDate: settingDate("04-10-2026"),
	}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var patched TopicResponse
	decodeBody(t, resp, &patched)
	require.Equal(t, "04-10-2026", patched.TargetDate)
	require.Equal(t, "Health & fitness", patched.Category, "unspecified fields are unchanged")

	// Clear the target date by sending an empty string.
	resp = app.doJSON(t, http.MethodPatch, "/api/v1/topics/"+created.ID, UpdateTopicRequest{
		TargetDate: clearingDate(),
	}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var cleared TopicResponse
	decodeBody(t, resp, &cleared)
	require.Empty(t, cleared.TargetDate, "empty string clears the target date")

	// An invalid date format is rejected.
	bad := "2026-12-31"
	resp = app.doJSON(t, http.MethodPost, "/api/v1/topics", CreateTopicRequest{
		Title: "Bad date", TargetDate: bad,
	}, token)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	resp.Body.Close()
}

func TestTopicFlow_Unauthenticated(t *testing.T) {
	app := startApp(t)
	resp := app.doJSON(t, http.MethodGet, "/api/v1/topics", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestTopicFlow_Validation(t *testing.T) {
	app := startApp(t)
	token := app.registerAndLogin(t, "val@example.com")

	// Missing title -> 422.
	resp := app.doJSON(t, http.MethodPost, "/api/v1/topics", CreateTopicRequest{Description: "no title"}, token)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	resp.Body.Close()

	// Out-of-range year via raw query on list -> 422.
	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics?year=1000", nil, token)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	resp.Body.Close()

	// Non-numeric year -> 422.
	resp = app.doJSON(t, http.MethodGet, "/api/v1/topics?year=abc", nil, token)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	resp.Body.Close()
}

// TestTopicFlow_YearScoping confirms the list defaults to the current year and
// ?year= selects another.
func TestTopicFlow_YearScoping(t *testing.T) {
	app := startApp(t)
	token := app.registerAndLogin(t, "years@example.com")

	current := domain.CurrentYear()
	past := current - 1

	for _, tc := range []struct {
		title string
		year  int
	}{{"this year", current}, {"last year", past}} {
		resp := app.doJSON(t, http.MethodPost, "/api/v1/topics", CreateTopicRequest{Title: tc.title, Year: tc.year}, token)
		require.Equal(t, http.StatusCreated, resp.StatusCode, fmt.Sprintf("create %s", tc.title))
		resp.Body.Close()
	}

	resp := app.doJSON(t, http.MethodGet, "/api/v1/topics", nil, token)
	var def []TopicResponse
	decodeBody(t, resp, &def)
	require.Len(t, def, 1)
	require.Equal(t, current, def[0].Year)

	resp = app.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/topics?year=%d", past), nil, token)
	var old []TopicResponse
	decodeBody(t, resp, &old)
	require.Len(t, old, 1)
	require.Equal(t, past, old[0].Year)
}
