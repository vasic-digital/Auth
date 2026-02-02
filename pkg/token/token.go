// Package token provides token abstractions and utilities for authentication
// systems, including a generic Token interface, Claims map type, and an
// in-memory token store with TTL support.
package token

import (
	"fmt"
	"sync"
	"time"
)

// Token represents a generic authentication token with access and refresh
// capabilities, expiration checking, and refresh threshold detection.
type Token interface {
	// AccessToken returns the access token string.
	AccessToken() string

	// RefreshToken returns the refresh token string.
	RefreshToken() string

	// ExpiresAt returns the expiration time of the token.
	ExpiresAt() time.Time

	// IsExpired returns true if the token has expired.
	IsExpired() bool

	// NeedsRefresh returns true if the token should be refreshed
	// based on the provided threshold duration before expiration.
	NeedsRefresh(threshold time.Duration) bool
}

// Claims is a map of claim key-value pairs extracted from a token.
// It provides helper methods for common JWT claim fields.
type Claims map[string]interface{}

// Subject returns the "sub" claim as a string, or empty if not present.
func (c Claims) Subject() string {
	return c.getString("sub")
}

// Issuer returns the "iss" claim as a string, or empty if not present.
func (c Claims) Issuer() string {
	return c.getString("iss")
}

// Audience returns the "aud" claim as a string, or empty if not present.
func (c Claims) Audience() string {
	return c.getString("aud")
}

// ExpiresAt returns the "exp" claim as a time.Time.
// Returns zero time if the claim is not present or cannot be parsed.
func (c Claims) ExpiresAt() time.Time {
	return c.getTime("exp")
}

// IssuedAt returns the "iat" claim as a time.Time.
// Returns zero time if the claim is not present or cannot be parsed.
func (c Claims) IssuedAt() time.Time {
	return c.getTime("iat")
}

// Get returns the value for a given key, or nil if not present.
func (c Claims) Get(key string) interface{} {
	return c[key]
}

// GetString returns the string value for a given key, or empty if not
// present or not a string.
func (c Claims) GetString(key string) string {
	return c.getString(key)
}

func (c Claims) getString(key string) string {
	v, ok := c[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func (c Claims) getTime(key string) time.Time {
	v, ok := c[key]
	if !ok {
		return time.Time{}
	}
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0)
	case int64:
		return time.Unix(t, 0)
	case time.Time:
		return t
	default:
		return time.Time{}
	}
}

// Store provides an interface for token storage with get, set, delete,
// and revoke operations.
type Store interface {
	// Get retrieves a token by its key. Returns an error if not found
	// or expired.
	Get(key string) (Token, error)

	// Set stores a token with the given key and optional TTL.
	// A zero TTL means the token does not expire from the store.
	Set(key string, token Token, ttl time.Duration) error

	// Delete removes a token by its key.
	Delete(key string) error

	// Revoke marks a token as revoked. Revoked tokens cannot be
	// retrieved via Get.
	Revoke(key string) error
}

// storeEntry holds a token with its expiration metadata in the store.
type storeEntry struct {
	token     Token
	expiresAt time.Time
	revoked   bool
}

// InMemoryStore is a thread-safe in-memory implementation of Store
// with TTL support and automatic cleanup of expired entries.
type InMemoryStore struct {
	mu      sync.RWMutex
	entries map[string]*storeEntry
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		entries: make(map[string]*storeEntry),
	}
}

// Get retrieves a token by key. Returns an error if the token is not
// found, has been revoked, or has expired from the store.
func (s *InMemoryStore) Get(key string) (Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[key]
	if !ok {
		return nil, fmt.Errorf("token not found: %s", key)
	}

	if entry.revoked {
		return nil, fmt.Errorf("token has been revoked: %s", key)
	}

	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return nil, fmt.Errorf("token has expired from store: %s", key)
	}

	return entry.token, nil
}

// Set stores a token with the given key. If ttl is zero, the entry
// does not expire from the store (the token itself may still expire).
func (s *InMemoryStore) Set(
	key string, token Token, ttl time.Duration,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := &storeEntry{
		token: token,
	}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}

	s.entries[key] = entry
	return nil
}

// Delete removes a token entry by key.
func (s *InMemoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entries[key]; !ok {
		return fmt.Errorf("token not found: %s", key)
	}

	delete(s.entries, key)
	return nil
}

// Revoke marks a token as revoked. The entry remains in the store
// but Get will return an error for revoked tokens.
func (s *InMemoryStore) Revoke(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[key]
	if !ok {
		return fmt.Errorf("token not found: %s", key)
	}

	entry.revoked = true
	return nil
}

// Cleanup removes all expired and revoked entries from the store.
func (s *InMemoryStore) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	removed := 0
	for key, entry := range s.entries {
		if entry.revoked ||
			(!entry.expiresAt.IsZero() && now.After(entry.expiresAt)) {
			delete(s.entries, key)
			removed++
		}
	}
	return removed
}

// Len returns the number of entries currently in the store.
func (s *InMemoryStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// SimpleToken is a basic implementation of the Token interface.
type SimpleToken struct {
	accessToken  string
	refreshToken string
	expiresAt    time.Time
}

// NewSimpleToken creates a new SimpleToken.
func NewSimpleToken(
	accessToken, refreshToken string, expiresAt time.Time,
) *SimpleToken {
	return &SimpleToken{
		accessToken:  accessToken,
		refreshToken: refreshToken,
		expiresAt:    expiresAt,
	}
}

// AccessToken returns the access token string.
func (t *SimpleToken) AccessToken() string {
	return t.accessToken
}

// RefreshToken returns the refresh token string.
func (t *SimpleToken) RefreshToken() string {
	return t.refreshToken
}

// ExpiresAt returns the expiration time.
func (t *SimpleToken) ExpiresAt() time.Time {
	return t.expiresAt
}

// IsExpired returns true if the token has expired.
func (t *SimpleToken) IsExpired() bool {
	if t.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(t.expiresAt)
}

// NeedsRefresh returns true if the token is within the threshold
// of expiration.
func (t *SimpleToken) NeedsRefresh(threshold time.Duration) bool {
	if t.expiresAt.IsZero() {
		return false
	}
	return time.Until(t.expiresAt) < threshold
}
