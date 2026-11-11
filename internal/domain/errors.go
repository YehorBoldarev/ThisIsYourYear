// Package domain holds business-level entities and the errors that express
// domain outcomes independently of any transport or storage technology. The
// rest of the application speaks in these terms; adapters (HTTP, Mongo) translate
// to and from them at the edges.
package domain

import "errors"

// Sentinel domain errors. They are intended to be wrapped with context using
// fmt.Errorf("...: %w", ...) and inspected with errors.Is or the helpers below.
var (
	// ErrNotFound indicates a requested resource does not exist.
	ErrNotFound = errors.New("resource not found")
	// ErrConflict indicates a write violates an invariant such as a uniqueness
	// constraint (e.g. a duplicate email).
	ErrConflict = errors.New("resource conflict")
	// ErrValidation indicates input failed a business or format rule.
	ErrValidation = errors.New("validation failed")
	// ErrUnauthenticated indicates the request lacks a valid credential (e.g. a
	// missing, malformed, or expired access token).
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrInvalidCredentials indicates an authentication attempt failed because the
	// supplied email/password did not match. It is deliberately generic so callers
	// cannot distinguish "unknown email" from "wrong password" (no user enumeration).
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// FieldError is a validation failure tied to one named input field, raised by a
// business rule (as opposed to request-shape validation in the transport layer).
// It unwraps to ErrValidation, and the HTTP adapter renders Field/Message in the
// error envelope's per-field map so a form can show it next to the right input.
type FieldError struct {
	Field   string
	Message string
}

// Error implements error.
func (e *FieldError) Error() string { return e.Field + ": " + e.Message }

// Unwrap makes errors.Is(err, ErrValidation) true for a FieldError.
func (e *FieldError) Unwrap() error { return ErrValidation }

// IsNotFound reports whether err is or wraps ErrNotFound.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// IsConflict reports whether err is or wraps ErrConflict.
func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }

// IsValidation reports whether err is or wraps ErrValidation.
func IsValidation(err error) bool { return errors.Is(err, ErrValidation) }

// IsUnauthenticated reports whether err is or wraps ErrUnauthenticated.
func IsUnauthenticated(err error) bool { return errors.Is(err, ErrUnauthenticated) }

// IsInvalidCredentials reports whether err is or wraps ErrInvalidCredentials.
func IsInvalidCredentials(err error) bool { return errors.Is(err, ErrInvalidCredentials) }
