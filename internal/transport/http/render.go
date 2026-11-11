package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"kibou/thisisyouryear/internal/domain"
)

// maxRequestBodyBytes caps request bodies to a sane size so a client cannot make
// the server buffer unbounded input.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// errorBody is the stable JSON error envelope returned to clients:
//
//	{ "error": { "code": "...", "message": "...", "fields": { ... } } }
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// decodeJSON reads a single JSON object from the request body into dst. Unknown
// fields and oversized or malformed bodies are rejected as validation errors so
// the caller can surface a 422 uniformly.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return &ValidationError{msg: "request body is not valid JSON"}
	}
	return nil
}

// respondError is the single place that maps a domain error to an HTTP status
// and response body. Handlers and middleware translate every failure here, so
// status-code policy lives in exactly one location and internal error details
// never leak to clients.
func respondError(w http.ResponseWriter, logger *slog.Logger, r *http.Request, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "internal server error"
	var fields map[string]string

	var ve *ValidationError
	var fe *domain.FieldError
	switch {
	case errors.As(err, &ve):
		status, code, message, fields = http.StatusUnprocessableEntity, "validation_error", ve.Error(), ve.Fields
	case errors.As(err, &fe):
		status, code, message = http.StatusUnprocessableEntity, "validation_error", "one or more fields are invalid"
		fields = map[string]string{fe.Field: fe.Message}
	case domain.IsValidation(err):
		status, code, message = http.StatusUnprocessableEntity, "validation_error", "validation failed"
	case domain.IsInvalidCredentials(err), domain.IsUnauthenticated(err):
		status, code, message = http.StatusUnauthorized, "unauthorized", "invalid or missing credentials"
	case domain.IsNotFound(err):
		status, code, message = http.StatusNotFound, "not_found", "resource not found"
	case domain.IsConflict(err):
		status, code, message = http.StatusConflict, "conflict", "resource already exists"
	default:
		// Unclassified: a real server fault. Log the detail; tell the client nothing.
		if logger != nil {
			logger.LogAttrs(r.Context(), slog.LevelError, "unhandled request error",
				slog.String("path", r.URL.Path),
				slog.Any("error", err),
			)
		}
	}

	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message, Fields: fields}})
}
