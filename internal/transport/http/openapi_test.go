package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"kibou/thisisyouryear/api"
)

// TestOpenAPISpec_IsWellFormed guards the checked-in contract against rotting
// into invalid YAML and confirms it describes the endpoints it should.
func TestOpenAPISpec_IsWellFormed(t *testing.T) {
	require.NotEmpty(t, api.OpenAPISpec, "embedded OpenAPI spec must not be empty")

	var doc struct {
		OpenAPI string         `yaml:"openapi"`
		Paths   map[string]any `yaml:"paths"`
	}
	require.NoError(t, yaml.Unmarshal(api.OpenAPISpec, &doc), "spec must be valid YAML")

	require.True(t, strings.HasPrefix(doc.OpenAPI, "3."), "expected OpenAPI 3.x, got %q", doc.OpenAPI)
	require.NotEmpty(t, doc.Paths, "spec must document paths")

	// A representative sample across the surface must be present.
	for _, p := range []string{
		"/api/v1/auth/login",
		"/api/v1/me",
		"/api/v1/topics",
		"/api/v1/topics/{id}",
		"/api/v1/goals/{id}",
		"/api/v1/dashboard",
		"/healthz",
	} {
		require.Contains(t, doc.Paths, p, "spec should document %s", p)
	}
}

// TestOpenAPIHandler serves the spec and checks the response shape.
func TestOpenAPIHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	openapiHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/yaml")
	require.Equal(t, api.OpenAPISpec, rec.Body.Bytes())
}
