// Package gin provides Gin framework adapters for digital.vasic.auth.
package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"digital.vasic.auth/pkg/accesstoken"
)

// Context keys for storing auth information.
const (
	CtxUserID      = "user_id"
	CtxAccessToken = "access_token"
)

// Config holds configuration for the auth Gin middleware.
type Config struct {
	// HeaderName is the HTTP header to read the access token from.
	HeaderName string
	// QueryParam is the query parameter to read the access token from (fallback).
	QueryParam string
}

// DefaultConfig returns a Config with sensible default conventions.
func DefaultConfig() *Config {
	return &Config{
		HeaderName: "X-Access-Token",
		QueryParam: "access",
	}
}

// AccessTokenAuth creates a Gin middleware that requires a valid access token.
// It reads the token from the configured header or query parameter, validates it
// against the store, and sets CtxUserID and CtxAccessToken in the Gin context.
func AccessTokenAuth(store accesstoken.Store, cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(cfg.HeaderName)
		if token == "" {
			token = c.Query(cfg.QueryParam)
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "access token required"})
			return
		}

		info, err := store.Validate(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired access token"})
			return
		}

		c.Set(CtxUserID, info.UserID)
		c.Set(CtxAccessToken, token)
		c.Next()
	}
}

// OptionalAuth creates a Gin middleware that validates access tokens if present
// but does not fail if missing.
func OptionalAuth(store accesstoken.Store, cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(cfg.HeaderName)
		if token == "" {
			token = c.Query(cfg.QueryParam)
		}
		if token != "" {
			if info, err := store.Validate(token); err == nil {
				c.Set(CtxUserID, info.UserID)
				c.Set(CtxAccessToken, token)
			}
		}
		c.Next()
	}
}
