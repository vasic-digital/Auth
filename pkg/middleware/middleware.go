// Package middleware provides HTTP authentication middleware for Bearer
// token validation, API key header validation, scope checking, and
// middleware chaining.
package middleware

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// ClaimsKey is the context key for storing validated claims.
	ClaimsKey contextKey = "auth_claims"

	// ScopesKey is the context key for storing validated scopes.
	ScopesKey contextKey = "auth_scopes"

	// APIKeyKey is the context key for storing the validated API key.
	APIKeyKey contextKey = "auth_api_key"
)

// Middleware is a function that wraps an http.Handler with
// authentication logic.
type Middleware func(http.Handler) http.Handler

// TokenValidator validates a token string and returns claims.
type TokenValidator interface {
	// ValidateToken validates the token and returns claims as a map.
	ValidateToken(token string) (map[string]interface{}, error)
}

// APIKeyValidator validates an API key string.
type APIKeyValidator interface {
	// ValidateKey validates the API key and returns scopes.
	ValidateKey(key string) ([]string, error)
}

// BearerToken creates middleware that extracts and validates a Bearer
// token from the Authorization header. On success, the claims are
// stored in the request context under ClaimsKey.
func BearerToken(validator TokenValidator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w,
					`{"error":"missing authorization header"}`,
					http.StatusUnauthorized,
				)
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w,
					`{"error":"invalid authorization scheme, expected Bearer"}`,
					http.StatusUnauthorized,
				)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == "" {
				http.Error(w,
					`{"error":"empty bearer token"}`,
					http.StatusUnauthorized,
				)
				return
			}

			claims, err := validator.ValidateToken(tokenStr)
			if err != nil {
				http.Error(w,
					`{"error":"invalid token"}`,
					http.StatusUnauthorized,
				)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)

			// Extract scopes if present. After a JWT round-trip, a claim
			// originally set as []string comes back as []interface{} (JSON
			// doesn't preserve typed arrays), so we must handle both.
			if scopes, ok := claims["scopes"]; ok {
				scopeList := coerceScopes(scopes)
				if len(scopeList) > 0 {
					ctx = context.WithValue(ctx, ScopesKey, scopeList)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyHeader creates middleware that validates an API key from the
// specified HTTP header. On success, the scopes are stored in the
// request context under ScopesKey and the key under APIKeyKey.
func APIKeyHeader(
	validator APIKeyValidator, headerName string,
) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(headerName)
			if key == "" {
				http.Error(w,
					`{"error":"missing API key header"}`,
					http.StatusUnauthorized,
				)
				return
			}

			scopes, err := validator.ValidateKey(key)
			if err != nil {
				http.Error(w,
					`{"error":"invalid API key"}`,
					http.StatusUnauthorized,
				)
				return
			}

			ctx := context.WithValue(r.Context(), APIKeyKey, key)
			ctx = context.WithValue(ctx, ScopesKey, scopes)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireScopes creates middleware that checks the request context
// for the required scopes. This middleware should be placed after
// BearerToken or APIKeyHeader in the chain.
func RequireScopes(required ...string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopesVal := r.Context().Value(ScopesKey)
			if scopesVal == nil {
				http.Error(w,
					`{"error":"no scopes in context"}`,
					http.StatusForbidden,
				)
				return
			}

			scopes, ok := scopesVal.([]string)
			if !ok {
				http.Error(w,
					`{"error":"invalid scopes in context"}`,
					http.StatusForbidden,
				)
				return
			}

			scopeSet := make(map[string]bool, len(scopes))
			for _, s := range scopes {
				scopeSet[s] = true
			}

			for _, req := range required {
				if !scopeSet[req] {
					http.Error(w,
						`{"error":"insufficient scopes"}`,
						http.StatusForbidden,
					)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Chain combines multiple middleware into a single middleware.
// Middleware are applied in order, so the first middleware in the
// list is the outermost wrapper.
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// ClaimsFromContext extracts claims from the request context.
// Returns nil if no claims are present.
func ClaimsFromContext(ctx context.Context) map[string]interface{} {
	claims, _ := ctx.Value(ClaimsKey).(map[string]interface{})
	return claims
}

// ScopesFromContext extracts scopes from the request context.
// Returns nil if no scopes are present.
func ScopesFromContext(ctx context.Context) []string {
	scopes, _ := ctx.Value(ScopesKey).([]string)
	return scopes
}

// coerceScopes normalizes a scopes claim into []string. JWT round-trips
// turn []string into []interface{} because JSON has no typed arrays, so
// the token validator must accept both. All other types yield nil —
// callers that want RFC 6749 space-separated string-scope support
// should pre-parse before putting the claim in.
func coerceScopes(v interface{}) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []interface{}:
		out := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

// APIKeyFromContext extracts the API key from the request context.
// Returns empty string if no API key is present.
func APIKeyFromContext(ctx context.Context) string {
	key, _ := ctx.Value(APIKeyKey).(string)
	return key
}
