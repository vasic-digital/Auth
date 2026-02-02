package jwt

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-jwt-testing"

func newTestManager() *Manager {
	return NewManager(DefaultConfig(testSecret))
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("my-secret")
	assert.Equal(t, gojwt.SigningMethodHS256, cfg.SigningMethod)
	assert.Equal(t, []byte("my-secret"), cfg.Secret)
	assert.Equal(t, time.Hour, cfg.Expiration)
}

func TestManager_Create(t *testing.T) {
	tests := []struct {
		name   string
		claims map[string]interface{}
	}{
		{
			name:   "with claims",
			claims: map[string]interface{}{"sub": "user-1", "role": "admin"},
		},
		{
			name:   "nil claims",
			claims: nil,
		},
		{
			name:   "empty claims",
			claims: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestManager()
			tokenStr, err := m.Create(tt.claims)
			require.NoError(t, err)
			assert.NotEmpty(t, tokenStr)
		})
	}
}

func TestManager_Create_WithIssuer(t *testing.T) {
	cfg := DefaultConfig(testSecret)
	cfg.Issuer = "test-issuer"
	m := NewManager(cfg)

	tokenStr, err := m.Create(map[string]interface{}{"sub": "user-1"})
	require.NoError(t, err)

	tok, err := m.Validate(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "test-issuer", tok.Claims["iss"])
}

func TestManager_Validate_Valid(t *testing.T) {
	m := newTestManager()

	claims := map[string]interface{}{
		"sub":  "user-123",
		"role": "admin",
	}
	tokenStr, err := m.Create(claims)
	require.NoError(t, err)

	tok, err := m.Validate(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "user-123", tok.Claims["sub"])
	assert.Equal(t, "admin", tok.Claims["role"])
	assert.False(t, tok.ExpiresAt.IsZero())
	assert.False(t, tok.IssuedAt.IsZero())
	assert.Equal(t, tokenStr, tok.Raw)
}

func TestManager_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name       string
		tokenStr   string
		errContain string
	}{
		{
			name:       "empty string",
			tokenStr:   "",
			errContain: "failed to parse",
		},
		{
			name:       "garbage",
			tokenStr:   "not-a-valid-jwt",
			errContain: "failed to parse",
		},
		{
			name:       "wrong secret",
			tokenStr:   createTokenWithSecret("wrong-secret"),
			errContain: "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestManager()
			_, err := m.Validate(tt.tokenStr)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContain)
		})
	}
}

func TestManager_Validate_Expired(t *testing.T) {
	cfg := DefaultConfig(testSecret)
	cfg.Expiration = -time.Hour // Already expired
	m := NewManager(cfg)

	tokenStr, err := m.Create(map[string]interface{}{"sub": "user-1"})
	require.NoError(t, err)

	_, err = m.Validate(tokenStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestManager_Validate_WrongSigningMethod(t *testing.T) {
	// Create token with HS384
	cfg384 := &Config{
		SigningMethod: gojwt.SigningMethodHS384,
		Secret:        []byte(testSecret),
		Expiration:    time.Hour,
	}
	m384 := NewManager(cfg384)
	tokenStr, err := m384.Create(map[string]interface{}{"sub": "user-1"})
	require.NoError(t, err)

	// Try to validate with HS256
	m256 := newTestManager()
	_, err = m256.Validate(tokenStr)
	assert.Error(t, err)
}

func TestManager_Refresh(t *testing.T) {
	m := newTestManager()

	claims := map[string]interface{}{
		"sub":  "user-123",
		"role": "admin",
	}
	original, err := m.Create(claims)
	require.NoError(t, err)

	// Wait for at least 1 second to ensure different iat/exp
	time.Sleep(1100 * time.Millisecond)

	refreshed, err := m.Refresh(original)
	require.NoError(t, err)
	assert.NotEqual(t, original, refreshed)

	// Verify refreshed token has same custom claims
	tok, err := m.Validate(refreshed)
	require.NoError(t, err)
	assert.Equal(t, "user-123", tok.Claims["sub"])
	assert.Equal(t, "admin", tok.Claims["role"])
}

func TestManager_Refresh_InvalidToken(t *testing.T) {
	m := newTestManager()

	_, err := m.Refresh("invalid-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot refresh invalid token")
}

func createTokenWithSecret(secret string) string {
	m := NewManager(DefaultConfig(secret))
	tok, _ := m.Create(map[string]interface{}{"sub": "test"})
	return tok
}
