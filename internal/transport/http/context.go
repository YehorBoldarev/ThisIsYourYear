package http

import "context"

// contextKey is an unexported type for context keys defined in this package, so
// they cannot collide with keys from other packages.
type contextKey int

const userIDKey contextKey = iota

// withUserID returns a copy of ctx carrying the authenticated user's id. It is
// set by the auth middleware after a token is verified.
func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserID returns the authenticated user's id from ctx, if the request passed
// through the auth middleware.
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}
