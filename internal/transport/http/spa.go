package http

import (
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"kibou/thisisyouryear/internal/domain"
)

// contentSecurityPolicy locks the served document to same-origin resources. The
// SPA loads its JS/CSS/fonts from this origin and calls only /api on the same
// host, so 'self' suffices. 'unsafe-inline' is granted for styles only (the UI
// uses inline style attributes); scripts stay strict 'self' because the theme
// bootstrap was moved to an external /theme-init.js file.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

// setSecurityHeaders applies the baseline hardening headers for SPA responses.
func setSecurityHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "same-origin")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Content-Security-Policy", contentSecurityPolicy)
}

// newSPAHandler serves the embedded single-page app: a static file when one
// exists, otherwise index.html so client-side routes (e.g. /achievements) deep-
// link and reload. It is mounted as the router's NotFound handler, so it runs
// only for requests the API/ops routes did not match.
//
// Requests under /api/ never render the app shell — they get the JSON 404
// envelope — and only GET/HEAD can serve HTML.
func newSPAHandler(fsys fs.FS, logger *slog.Logger) http.HandlerFunc {
	fileServer := http.FileServerFS(fsys)

	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			respondError(w, logger, r, domain.ErrNotFound)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			respondError(w, logger, r, domain.ErrNotFound)
			return
		}

		setSecurityHeaders(w)

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			serveIndex(w, fsys)
			return
		}
		if f, err := fsys.Open(name); err == nil {
			_ = f.Close()
			// Vite emits content-hashed bundles under assets/, so they can be
			// cached indefinitely; a new deploy ships new filenames.
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		// Unknown, non-API path → a client-side route → serve the app shell.
		serveIndex(w, fsys)
	}
}

// serveIndex writes index.html with a no-cache header so browsers always re-fetch
// the shell (and thus the current hashed asset references) after a deploy.
func serveIndex(w http.ResponseWriter, fsys fs.FS) {
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(index)
}
