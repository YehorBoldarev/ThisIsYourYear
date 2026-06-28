package http

import (
	"context"
	"encoding/json"
	"net/http"
)

// Pinger reports whether a downstream dependency (e.g. MongoDB) is reachable.
// The health handlers depend on this narrow interface rather than a concrete
// client, which keeps them trivially unit-testable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// healthHandler answers liveness probes. It reports that the process itself is
// running and able to serve, without checking any external dependency. A live
// but not-yet-ready process should still return 200 here.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readinessHandler answers readiness probes. It reports whether the service can
// currently serve real traffic, which requires its dependencies to be reachable.
func readinessHandler(pinger Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pinger.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "unavailable",
				"reason": "database unreachable",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

// writeJSON serializes v as JSON with the given status code. Encoding errors are
// not recoverable at this point (headers are already written), so they are
// silently ignored; the small payloads here cannot realistically fail to encode.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
