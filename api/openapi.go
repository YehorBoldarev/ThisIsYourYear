// Package api holds the checked-in HTTP contract for the ThisIsYourYear service
// and embeds it so the running server can publish it. The OpenAPI document in
// openapi.yaml is the single source of truth for the API's shape; keep it in
// step with the transport DTOs.
package api

import _ "embed"

// OpenAPISpec is the raw bytes of the OpenAPI 3.1 document, embedded at build
// time so the binary is self-contained (no file-path dependency at runtime).
//
//go:embed openapi.yaml
var OpenAPISpec []byte
