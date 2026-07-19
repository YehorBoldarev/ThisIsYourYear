//go:build !embedui

// Package webui exposes the built single-page app as a filesystem the HTTP layer
// can serve. Without the `embedui` build tag the binary is API-only — the default
// for local development and backend CI, which must not require a built frontend.
package webui

import "io/fs"

// Enabled reports whether the SPA was compiled in. It is false here: no UI is
// embedded, so the router serves only the API and operational endpoints.
const Enabled = false

// FS returns nil when no SPA is embedded.
func FS() fs.FS { return nil }
