// Package jwt provides JWT token creation, validation, and refresh
// functionality using configurable signing methods and secrets.
package jwt

import (
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// Config holds JWT configuration for token creation and validation.
type Config struct {
	// SigningMethod is the algorithm used to sign tokens (e.g., HS256).
	SigningMethod gojwt.SigningMethod

	// Secret is the key used to sign and verify tokens.
	Secret []byte

	// Expiration is the default token lifetime.
	Expiration time.Duration

	// Issuer is the optional issuer claim set on created tokens.
	Issuer string
}

// DefaultConfig returns a Config with sensible defaults: HS256 signing
// and 1-hour expiration.
func DefaultConfig(secret string) *Config {
	return &Config{
		SigningMethod: gojwt.SigningMethodHS256,
		Secret:        []byte(secret),
		Expiration:    time.Hour,
	}
}

// Token represents a parsed JWT token with its claims and metadata.
type Token struct {
	// Claims contains the token's claim key-value pairs.
	Claims map[string]interface{}

	// ExpiresAt is the token's expiration time.
	ExpiresAt time.Time

	// IssuedAt is the time the token was issued.
	IssuedAt time.Time

	// Raw is the original signed token string.
	Raw string
}

// Parser defines the interface for parsing JWT tokens.
// This allows dependency injection for testing.
type Parser interface {
	Parse(tokenString string, keyFunc gojwt.Keyfunc) (*gojwt.Token, error)
}

// defaultParser wraps gojwt.Parse as the default implementation.
type defaultParser struct{}

func (p *defaultParser) Parse(
	tokenString string, keyFunc gojwt.Keyfunc,
) (*gojwt.Token, error) {
	return gojwt.Parse(tokenString, keyFunc)
}

// Manager handles JWT token operations using the provided Config.
type Manager struct {
	config *Config
	parser Parser
}

// NewManager creates a new JWT Manager with the given configuration.
func NewManager(config *Config) *Manager {
	return &Manager{config: config, parser: &defaultParser{}}
}

// SetParser sets a custom parser for testing purposes.
func (m *Manager) SetParser(p Parser) {
	m.parser = p
}

// Create generates a signed JWT string from the provided claims.
// Standard claims (exp, iat, iss) are set automatically based on
// the configuration.
func (m *Manager) Create(claims map[string]interface{}) (string, error) {
	if claims == nil {
		claims = make(map[string]interface{})
	}

	now := time.Now()
	mapClaims := gojwt.MapClaims{}

	// Copy user claims
	for k, v := range claims {
		mapClaims[k] = v
	}

	// Set standard claims
	mapClaims["iat"] = now.Unix()
	mapClaims["exp"] = now.Add(m.config.Expiration).Unix()

	if m.config.Issuer != "" {
		mapClaims["iss"] = m.config.Issuer
	}

	token := gojwt.NewWithClaims(m.config.SigningMethod, mapClaims)

	signed, err := token.SignedString(m.config.Secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, nil
}

// Validate parses and validates a JWT token string. Returns the
// decoded Token with claims on success, or an error if the token
// is invalid, expired, or cannot be verified.
func (m *Manager) Validate(tokenString string) (*Token, error) {
	parsed, err := m.parser.Parse(tokenString, func(t *gojwt.Token) (interface{}, error) {
		if t.Method.Alg() != m.config.SigningMethod.Alg() {
			return nil, fmt.Errorf(
				"unexpected signing method: %v", t.Header["alg"],
			)
		}
		return m.config.Secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	mapClaims, ok := parsed.Claims.(gojwt.MapClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	result := &Token{
		Claims: make(map[string]interface{}),
		Raw:    tokenString,
	}

	for k, v := range mapClaims {
		result.Claims[k] = v
	}

	// Extract standard time claims
	if exp, ok := mapClaims["exp"].(float64); ok {
		result.ExpiresAt = time.Unix(int64(exp), 0)
	}
	if iat, ok := mapClaims["iat"].(float64); ok {
		result.IssuedAt = time.Unix(int64(iat), 0)
	}

	return result, nil
}

// Refresh validates the existing token and creates a new token with
// the same claims but fresh expiration. Returns an error if the
// original token is invalid.
func (m *Manager) Refresh(tokenString string) (string, error) {
	token, err := m.Validate(tokenString)
	if err != nil {
		return "", fmt.Errorf("cannot refresh invalid token: %w", err)
	}

	// Remove standard claims that will be regenerated
	claims := make(map[string]interface{})
	for k, v := range token.Claims {
		if k == "exp" || k == "iat" || k == "iss" {
			continue
		}
		claims[k] = v
	}

	return m.Create(claims)
}
