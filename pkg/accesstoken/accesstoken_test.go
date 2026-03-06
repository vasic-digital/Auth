package accesstoken

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_ReturnsHexString(t *testing.T) {
	token, err := Generate(32)
	require.NoError(t, err)

	// 32 bytes -> 64 hex characters
	assert.Len(t, token, 64)

	// Must be valid hex
	decoded, err := hex.DecodeString(token)
	require.NoError(t, err)
	assert.Len(t, decoded, 32)
}

func TestGenerate_UniqueTokens(t *testing.T) {
	token1, err := Generate(32)
	require.NoError(t, err)

	token2, err := Generate(32)
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2)
}

func TestMemoryStore_StoreAndValidate(t *testing.T) {
	store := NewMemoryStore()
	info := &TokenInfo{
		UserID:    "user-123",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Metadata:  map[string]string{"device": "mobile"},
	}

	store.Store("tok-abc", info)

	got, err := store.Validate("tok-abc")
	require.NoError(t, err)
	assert.Equal(t, "user-123", got.UserID)
	assert.Equal(t, "mobile", got.Metadata["device"])
}

func TestMemoryStore_ExpiredToken(t *testing.T) {
	store := NewMemoryStore()
	info := &TokenInfo{
		UserID:    "user-456",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour),
	}

	store.Store("tok-expired", info)

	_, err := store.Validate("tok-expired")
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestMemoryStore_InvalidToken(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.Validate("nonexistent-token")
	assert.ErrorIs(t, err, ErrTokenNotFound)
}

func TestMemoryStore_Revoke(t *testing.T) {
	store := NewMemoryStore()
	info := &TokenInfo{
		UserID:    "user-789",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	store.Store("tok-revoke", info)

	// Validate before revoke
	_, err := store.Validate("tok-revoke")
	require.NoError(t, err)

	// Revoke the token
	err = store.Revoke("tok-revoke")
	require.NoError(t, err)

	// Validate after revoke should fail
	_, err = store.Validate("tok-revoke")
	assert.ErrorIs(t, err, ErrTokenNotFound)
}

func TestMemoryStore_RevokeAllForUser(t *testing.T) {
	store := NewMemoryStore()
	userID := "user-bulk"

	// Store 3 tokens for the same user
	for i, tok := range []string{"tok-1", "tok-2", "tok-3"} {
		store.Store(tok, &TokenInfo{
			UserID:    userID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Duration(i+1) * time.Hour),
		})
	}

	// Store 1 token for a different user
	store.Store("tok-other", &TokenInfo{
		UserID:    "user-other",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	})

	count := store.RevokeAllForUser(userID)
	assert.Equal(t, 3, count)

	// All tokens for the target user should be gone
	for _, tok := range []string{"tok-1", "tok-2", "tok-3"} {
		_, err := store.Validate(tok)
		assert.ErrorIs(t, err, ErrTokenNotFound)
	}

	// Other user's token should still be valid
	got, err := store.Validate("tok-other")
	require.NoError(t, err)
	assert.Equal(t, "user-other", got.UserID)
}
