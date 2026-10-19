package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// UserContextKey is the key for storing user claims in context
	UserContextKey ContextKey = "user"
)

// AuthMiddleware checks for valid JWT token
func AuthMiddleware(authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Extract bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error": "invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Validate token
			claims, err := authService.ValidateToken(token)
			if err != nil {
				http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Store claims in context
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(requiredRoles ...models.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Comma-ok form: a plain type assertion would panic if the route was
			// accidentally registered without AuthMiddleware in front of it.
			claims, ok := r.Context().Value(UserContextKey).(*services.Claims)
			if !ok || claims == nil {
				http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Check if user role is in required roles
			userRole := models.UserRole(claims.Role)
			hasRole := false
			for _, role := range requiredRoles {
				if userRole == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, `{"error": "forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserClaims retrieves user claims from context
func GetUserClaims(r *http.Request) *services.Claims {
	claims, ok := r.Context().Value(UserContextKey).(*services.Claims)
	if !ok {
		return nil
	}
	return claims
}
