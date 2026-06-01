package auth

import (
	"context"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const userContextKey contextKey = "user"

// WithUser adds user claims to the context
func WithUser(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}

// GetUser retrieves user claims from the context
func GetUser(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(userContextKey).(*Claims)
	return claims, ok
}

// GetUserID retrieves the user ID from the context
func GetUserID(ctx context.Context) (uint, bool) {
	claims, ok := GetUser(ctx)
	if !ok {
		return 0, false
	}
	return claims.UserID, true
}

// GetUsername retrieves the username from the context
func GetUsername(ctx context.Context) (string, bool) {
	claims, ok := GetUser(ctx)
	if !ok {
		return "", false
	}
	return claims.Username, true
}

// IsAdmin checks if the user in context is an admin
func IsAdmin(ctx context.Context) (bool, bool) {
	claims, ok := GetUser(ctx)
	if !ok {
		return false, false
	}
	return claims.IsAdmin, true
}
