// Package tokenmanager provides thread-safe authentication token lifecycle
// management including storage, expiration tracking, and validation.
package tokenmanager

import (
	"fmt"
	"sync"
	"time"
)

// Storage defines the interface for secure token storage backends.
type Storage interface {
	// Store saves a value under the given key.
	Store(key, value string) error

	// Retrieve gets the value for the given key, or empty string if not found.
	Retrieve(key string) (string, error)

	// Delete removes the value for the given key.
	Delete(key string) error
}

// TokenInfo holds non-sensitive metadata about stored tokens for debugging.
type TokenInfo struct {
	HasAccessToken  bool
	HasRefreshToken bool
	IsExpired       bool
	ServiceName     string
	Timestamp       time.Time
}

// Manager handles token lifecycle for a specific service.
type Manager struct {
	serviceName string
	storage     Storage
	mu          sync.Mutex
}

// New creates a new token Manager for the given service.
func New(serviceName string, storage Storage) *Manager {
	return &Manager{
		serviceName: serviceName,
		storage:     storage,
	}
}

// StoreAccessToken saves an access token securely.
func (m *Manager) StoreAccessToken(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.storage.Store(m.serviceName+"_access_token", token)
}

// GetAccessToken retrieves the stored access token.
func (m *Manager) GetAccessToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.storage.Retrieve(m.serviceName + "_access_token")
}

// StoreRefreshToken saves a refresh token securely.
func (m *Manager) StoreRefreshToken(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.storage.Store(m.serviceName+"_refresh_token", token)
}

// GetRefreshToken retrieves the stored refresh token.
func (m *Manager) GetRefreshToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.storage.Retrieve(m.serviceName + "_refresh_token")
}

// StoreExpiration saves the token expiration time.
func (m *Manager) StoreExpiration(expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.storage.Store(m.serviceName+"_expires", fmt.Sprintf("%d", expiresAt.UnixMilli()))
}

// IsExpired returns true if the token has expired or no expiration is stored.
func (m *Manager) IsExpired() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isExpiredInternal()
}

func (m *Manager) isExpiredInternal() (bool, error) {
	val, err := m.storage.Retrieve(m.serviceName + "_expires")
	if err != nil {
		return true, err
	}
	if val == "" {
		return true, nil // No expiration stored, assume expired
	}
	var millis int64
	_, err = fmt.Sscanf(val, "%d", &millis)
	if err != nil {
		return true, fmt.Errorf("invalid expiration format: %w", err)
	}
	expiresAt := time.UnixMilli(millis)
	return time.Now().After(expiresAt), nil
}

// HasValidToken returns true if a non-empty access token exists and is not expired.
func (m *Manager) HasValidToken() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	token, err := m.storage.Retrieve(m.serviceName + "_access_token")
	if err != nil {
		return false, err
	}
	if token == "" {
		return false, nil
	}
	expired, err := m.isExpiredInternal()
	if err != nil {
		return false, err
	}
	return !expired, nil
}

// ClearTokens removes all tokens for this service.
func (m *Manager) ClearTokens() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	if err := m.storage.Delete(m.serviceName + "_access_token"); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := m.storage.Delete(m.serviceName + "_refresh_token"); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := m.storage.Delete(m.serviceName + "_expires"); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// StoreTokenInfo stores access token, optional refresh token, and optional expiration.
func (m *Manager) StoreTokenInfo(accessToken string, refreshToken string, expiresIn time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.storage.Store(m.serviceName+"_access_token", accessToken); err != nil {
		return err
	}
	if refreshToken != "" {
		if err := m.storage.Store(m.serviceName+"_refresh_token", refreshToken); err != nil {
			return err
		}
	}
	if expiresIn > 0 {
		expiresAt := time.Now().Add(expiresIn)
		if err := m.storage.Store(m.serviceName+"_expires", fmt.Sprintf("%d", expiresAt.UnixMilli())); err != nil {
			return err
		}
	}
	return nil
}

// GetTokenInfo returns non-sensitive metadata about the stored tokens.
func (m *Manager) GetTokenInfo() (*TokenInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	accessToken, _ := m.storage.Retrieve(m.serviceName + "_access_token")
	refreshToken, _ := m.storage.Retrieve(m.serviceName + "_refresh_token")
	expired, _ := m.isExpiredInternal()

	return &TokenInfo{
		HasAccessToken:  accessToken != "",
		HasRefreshToken: refreshToken != "",
		IsExpired:       expired,
		ServiceName:     m.serviceName,
		Timestamp:       time.Now(),
	}, nil
}

// ServiceName returns the service name this manager handles.
func (m *Manager) ServiceName() string {
	return m.serviceName
}
