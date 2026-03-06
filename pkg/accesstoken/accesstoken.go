// Package accesstoken provides access token generation, validation, and
// storage for session-based authentication. Tokens are cryptographically
// random hex-encoded strings with associated user metadata and expiration.
package accesstoken

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	// ErrTokenNotFound is returned when a token does not exist in
	// the store.
	ErrTokenNotFound = errors.New("token not found")

	// ErrTokenExpired is returned when a token exists but has passed
	// its expiration time.
	ErrTokenExpired = errors.New("token expired")
)

// TokenInfo holds metadata associated with an access token, including
// the owning user, creation and expiration times, and arbitrary metadata.
type TokenInfo struct {
	// UserID is the identifier of the user who owns this token.
	UserID string

	// CreatedAt is the time the token was created.
	CreatedAt time.Time

	// ExpiresAt is the time after which the token is no longer valid.
	ExpiresAt time.Time

	// Metadata holds arbitrary key-value pairs associated with the
	// token (e.g., device info, IP address).
	Metadata map[string]string
}

// Store provides an interface for access token storage backends with
// store, validate, revoke, and bulk revoke operations.
type Store interface {
	// Store saves a token with its associated info.
	Store(token string, info *TokenInfo)

	// Validate retrieves and validates a token. Returns ErrTokenNotFound
	// if the token does not exist, or ErrTokenExpired if it has expired.
	Validate(token string) (*TokenInfo, error)

	// Revoke removes a single token from the store.
	Revoke(token string) error

	// RevokeAllForUser removes all tokens belonging to the given user
	// and returns the number of tokens revoked.
	RevokeAllForUser(userID string) int
}

// Generate creates a cryptographically random access token of the
// specified byte length, returned as a hex-encoded string. The
// resulting string will be twice the byte length in characters.
func Generate(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// MemoryStore is a thread-safe in-memory implementation of Store.
type MemoryStore struct {
	mu     sync.RWMutex
	tokens map[string]*TokenInfo
}

// NewMemoryStore creates a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tokens: make(map[string]*TokenInfo)}
}

// Store saves a token with its associated info.
func (s *MemoryStore) Store(token string, info *TokenInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = info
}

// Validate retrieves and validates a token. Returns ErrTokenNotFound
// if the token does not exist, or ErrTokenExpired if it has expired.
func (s *MemoryStore) Validate(token string) (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.tokens[token]
	if !ok {
		return nil, ErrTokenNotFound
	}
	if time.Now().After(info.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	return info, nil
}

// Revoke removes a single token from the store.
func (s *MemoryStore) Revoke(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
	return nil
}

// RevokeAllForUser removes all tokens belonging to the given user
// and returns the number of tokens revoked.
func (s *MemoryStore) RevokeAllForUser(userID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for token, info := range s.tokens {
		if info.UserID == userID {
			delete(s.tokens, token)
			count++
		}
	}
	return count
}
