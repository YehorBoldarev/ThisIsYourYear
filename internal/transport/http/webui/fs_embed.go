//go:build embedui

// Package webui exposes the built single-page app as a filesystem the HTTP layer
// can serve. Building with `-tags embedui` compiles the contents of ./dist (the
// frontend build output, copied in by `make web-build`) into the binary.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Enabled reports that the SPA was compiled into this binary.
const Enabled = true

// FS returns the built SPA assets, rooted at the dist directory.
func FS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// dist is embedded at build time; a failure here is a build bug, not a
		// runtime condition.
		panic("webui: cannot root embedded dist: " + err.Error())
	}
	return sub
}
