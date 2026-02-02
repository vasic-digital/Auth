package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaims_Subject(t *testing.T) {
	tests := []struct {
		name     string
		claims   Claims
		expected string
	}{
		{
			name:     "present",
			claims:   Claims{"sub": "user-123"},
			expected: "user-123",
		},
		{
			name:     "missing",
			claims:   Claims{},
			expected: "",
		},
		{
			name:     "wrong type",
			claims:   Claims{"sub": 123},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.claims.Subject())
		})
	}
}

func TestClaims_Issuer(t *testing.T) {
	tests := []struct {
		name     string
		claims   Claims
		expected string
	}{
		{
			name:     "present",
			claims:   Claims{"iss": "auth-service"},
			expected: "auth-service",
		},
		{
			name:     "missing",
			claims:   Claims{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.claims.Issuer())
		})
	}
}

func TestClaims_Audience(t *testing.T) {
	tests := []struct {
		name     string
		claims   Claims
		expected string
	}{
		{
			name:     "present",
			claims:   Claims{"aud": "my-api"},
			expected: "my-api",
		},
		{
			name:     "missing",
			claims:   Claims{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.claims.Audience())
		})
	}
}

func TestClaims_ExpiresAt(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		claims   Claims
		expected time.Time
	}{
		{
			name:     "float64 unix",
			claims:   Claims{"exp": float64(now.Unix())},
			expected: time.Unix(now.Unix(), 0),
		},
		{
			name:     "int64 unix",
			claims:   Claims{"exp": now.Unix()},
			expected: time.Unix(now.Unix(), 0),
		},
		{
			name:     "time.Time",
			claims:   Claims{"exp": now},
			expected: now,
		},
		{
			name:     "missing",
			claims:   Claims{},
			expected: time.Time{},
		},
		{
			name:     "wrong type",
			claims:   Claims{"exp": "not-a-time"},
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.claims.ExpiresAt()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClaims_IssuedAt(t *testing.T) {
	now := time.Now()
	c := Claims{"iat": float64(now.Unix())}
	result := c.IssuedAt()
	assert.Equal(t, time.Unix(now.Unix(), 0), result)
}

func TestClaims_Get(t *testing.T) {
	c := Claims{"custom": "value", "number": 42}
	assert.Equal(t, "value", c.Get("custom"))
	assert.Equal(t, 42, c.Get("number"))
	assert.Nil(t, c.Get("missing"))
}

func TestClaims_GetString(t *testing.T) {
	c := Claims{"str": "hello", "num": 42}
	assert.Equal(t, "hello", c.GetString("str"))
	assert.Equal(t, "", c.GetString("num"))
	assert.Equal(t, "", c.GetString("missing"))
}

func TestSimpleToken_AccessToken(t *testing.T) {
	tok := NewSimpleToken("access", "refresh", time.Now().Add(time.Hour))
	assert.Equal(t, "access", tok.AccessToken())
}

func TestSimpleToken_RefreshToken(t *testing.T) {
	tok := NewSimpleToken("access", "refresh", time.Now().Add(time.Hour))
	assert.Equal(t, "refresh", tok.RefreshToken())
}

func TestSimpleToken_ExpiresAt(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	tok := NewSimpleToken("a", "r", exp)
	assert.Equal(t, exp, tok.ExpiresAt())
}

func TestSimpleToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(time.Hour),
			expected:  false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-time.Hour),
			expected:  true,
		},
		{
			name:      "zero time never expires",
			expiresAt: time.Time{},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewSimpleToken("a", "r", tt.expiresAt)
			assert.Equal(t, tt.expected, tok.IsExpired())
		})
	}
}

func TestSimpleToken_NeedsRefresh(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		threshold time.Duration
		expected  bool
	}{
		{
			name:      "within threshold",
			expiresAt: time.Now().Add(5 * time.Minute),
			threshold: 10 * time.Minute,
			expected:  true,
		},
		{
			name:      "outside threshold",
			expiresAt: time.Now().Add(30 * time.Minute),
			threshold: 10 * time.Minute,
			expected:  false,
		},
		{
			name:      "zero time never needs refresh",
			expiresAt: time.Time{},
			threshold: 10 * time.Minute,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewSimpleToken("a", "r", tt.expiresAt)
			assert.Equal(t, tt.expected, tok.NeedsRefresh(tt.threshold))
		})
	}
}

func TestInMemoryStore_SetAndGet(t *testing.T) {
	store := NewInMemoryStore()
	tok := NewSimpleToken("access", "refresh", time.Now().Add(time.Hour))

	err := store.Set("key1", tok, time.Hour)
	require.NoError(t, err)

	got, err := store.Get("key1")
	require.NoError(t, err)
	assert.Equal(t, "access", got.AccessToken())
}

func TestInMemoryStore_Get_NotFound(t *testing.T) {
	store := NewInMemoryStore()

	_, err := store.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryStore_Get_Expired(t *testing.T) {
	store := NewInMemoryStore()
	tok := NewSimpleToken("access", "refresh", time.Now().Add(time.Hour))

	// Set with very short TTL
	err := store.Set("key1", tok, time.Nanosecond)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(time.Millisecond)

	_, err = store.Get("key1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestInMemoryStore_Get_ZeroTTL(t *testing.T) {
	store := NewInMemoryStore()
	tok := NewSimpleToken("access", "refresh", time.Now().Add(time.Hour))

	err := store.Set("key1", tok, 0)
	require.NoError(t, err)

	got, err := store.Get("key1")
	require.NoError(t, err)
	assert.Equal(t, "access", got.AccessToken())
}

func TestInMemoryStore_Delete(t *testing.T) {
	store := NewInMemoryStore()
	tok := NewSimpleToken("access", "refresh", time.Now().Add(time.Hour))

	err := store.Set("key1", tok, time.Hour)
	require.NoError(t, err)

	err = store.Delete("key1")
	require.NoError(t, err)

	_, err = store.Get("key1")
	assert.Error(t, err)
}

func TestInMemoryStore_Delete_NotFound(t *testing.T) {
	store := NewInMemoryStore()

	err := store.Delete("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryStore_Revoke(t *testing.T) {
	store := NewInMemoryStore()
	tok := NewSimpleToken("access", "refresh", time.Now().Add(time.Hour))

	err := store.Set("key1", tok, time.Hour)
	require.NoError(t, err)

	err = store.Revoke("key1")
	require.NoError(t, err)

	_, err = store.Get("key1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")
}

func TestInMemoryStore_Revoke_NotFound(t *testing.T) {
	store := NewInMemoryStore()

	err := store.Revoke("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryStore_Cleanup(t *testing.T) {
	store := NewInMemoryStore()
	tok := NewSimpleToken("access", "refresh", time.Now().Add(time.Hour))

	// Add expired entry
	err := store.Set("expired", tok, time.Nanosecond)
	require.NoError(t, err)

	// Add revoked entry
	err = store.Set("revoked", tok, time.Hour)
	require.NoError(t, err)
	err = store.Revoke("revoked")
	require.NoError(t, err)

	// Add valid entry
	err = store.Set("valid", tok, time.Hour)
	require.NoError(t, err)

	time.Sleep(time.Millisecond)

	removed := store.Cleanup()
	assert.Equal(t, 2, removed)
	assert.Equal(t, 1, store.Len())

	_, err = store.Get("valid")
	require.NoError(t, err)
}

func TestInMemoryStore_Len(t *testing.T) {
	store := NewInMemoryStore()
	assert.Equal(t, 0, store.Len())

	tok := NewSimpleToken("a", "r", time.Now().Add(time.Hour))
	_ = store.Set("k1", tok, time.Hour)
	_ = store.Set("k2", tok, time.Hour)
	assert.Equal(t, 2, store.Len())
}
