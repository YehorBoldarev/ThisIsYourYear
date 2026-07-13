package http

import (
	"net/http"

	"kibou/thisisyouryear/api"
)

// openapiHandler serves the embedded OpenAPI contract as YAML. It is public
// (unauthenticated) so client tooling can fetch the spec without credentials.
func openapiHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(api.OpenAPISpec)
}
