// Package oauth provides generic OAuth2 credential management including
// file-based credential reading, HTTP-based token refresh, auto-refresh
// with caching and rate limiting, and expiration utilities.
package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Credentials represents a set of OAuth2 credentials with access token,
// refresh token, expiration, scopes, and arbitrary metadata.
type Credentials struct {
	// AccessToken is the OAuth2 access token.
	AccessToken string `json:"access_token"`

	// RefreshToken is the OAuth2 refresh token.
	RefreshToken string `json:"refresh_token,omitempty"`

	// ExpiresAt is the time when the access token expires.
	ExpiresAt time.Time `json:"expires_at"`

	// Scopes is the list of granted OAuth2 scopes.
	Scopes []string `json:"scopes,omitempty"`

	// Metadata holds additional provider-specific data.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// IsExpired returns true if the credentials have expired.
func (c *Credentials) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

// NeedsRefresh returns true if the credentials are within the
// given threshold of expiration.
func (c *Credentials) NeedsRefresh(threshold time.Duration) bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Until(c.ExpiresAt) < threshold
}

// CredentialReader provides an interface for reading OAuth2 credentials
// from a named provider.
type CredentialReader interface {
	// ReadCredentials reads OAuth2 credentials for the named provider.
	ReadCredentials(providerName string) (*Credentials, error)
}

// FileCredentialReader reads OAuth2 credentials from JSON files on disk.
// It maps provider names to file paths.
type FileCredentialReader struct {
	// paths maps provider names to credential file paths.
	paths map[string]string
}

// NewFileCredentialReader creates a new FileCredentialReader with the
// given provider-to-path mapping.
func NewFileCredentialReader(
	paths map[string]string,
) *FileCredentialReader {
	return &FileCredentialReader{paths: paths}
}

// ReadCredentials reads and parses OAuth2 credentials from the JSON
// file associated with the given provider name.
func (r *FileCredentialReader) ReadCredentials(
	providerName string,
) (*Credentials, error) {
	path, ok := r.paths[providerName]
	if !ok {
		return nil, fmt.Errorf(
			"no credential path configured for provider: %s",
			providerName,
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"credentials file not found for %s at %s: %w",
				providerName, path, err,
			)
		}
		return nil, fmt.Errorf(
			"failed to read credentials for %s: %w", providerName, err,
		)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf(
			"failed to parse credentials for %s: %w", providerName, err,
		)
	}

	if creds.AccessToken == "" {
		return nil, fmt.Errorf(
			"empty access token in credentials for %s", providerName,
		)
	}

	return &creds, nil
}

// TokenRefresher provides an interface for refreshing OAuth2 tokens.
type TokenRefresher interface {
	// Refresh exchanges a refresh token for new credentials using
	// the given token endpoint URL.
	Refresh(refreshToken, endpoint string) (*Credentials, error)
}

// RefreshResponse represents the standard OAuth2 token refresh response.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope,omitempty"`
}

// HTTPTokenRefresher implements TokenRefresher using HTTP POST requests
// to OAuth2 token endpoints.
type HTTPTokenRefresher struct {
	client   *http.Client
	clientID string
	params   map[string]string
}

// NewHTTPTokenRefresher creates a new HTTPTokenRefresher. The clientID
// is included in refresh requests if non-empty. Additional parameters
// can be specified via extraParams.
func NewHTTPTokenRefresher(
	client *http.Client,
	clientID string,
	extraParams map[string]string,
) *HTTPTokenRefresher {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &HTTPTokenRefresher{
		client:   client,
		clientID: clientID,
		params:   extraParams,
	}
}

// Refresh sends a token refresh request to the given endpoint and
// returns new credentials on success.
func (r *HTTPTokenRefresher) Refresh(
	refreshToken, endpoint string,
) (*Credentials, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("no refresh token provided")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	if r.clientID != "" {
		data.Set("client_id", r.clientID)
	}

	for k, v := range r.params {
		data.Set(k, v)
	}

	req, err := http.NewRequest(
		"POST", endpoint, strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"refresh failed with status %d: %s",
			resp.StatusCode, string(body),
		)
	}

	var refreshResp RefreshResponse
	if err := json.Unmarshal(body, &refreshResp); err != nil {
		return nil, fmt.Errorf(
			"failed to parse refresh response: %w", err,
		)
	}

	creds := &Credentials{
		AccessToken:  refreshResp.AccessToken,
		RefreshToken: refreshResp.RefreshToken,
	}

	if refreshResp.ExpiresIn > 0 {
		creds.ExpiresAt = time.Now().Add(
			time.Duration(refreshResp.ExpiresIn) * time.Second,
		)
	}

	if refreshResp.Scope != "" {
		creds.Scopes = strings.Split(refreshResp.Scope, " ")
	}

	// Keep existing refresh token if none was returned
	if creds.RefreshToken == "" {
		creds.RefreshToken = refreshToken
	}

	return creds, nil
}

// NeedsRefresh checks if a token with the given expiration time needs
// refreshing based on the threshold duration.
func NeedsRefresh(expiresAt time.Time, threshold time.Duration) bool {
	if expiresAt.IsZero() {
		return false
	}
	return time.Until(expiresAt) < threshold
}

// IsExpired checks if a token with the given expiration time has expired.
func IsExpired(expiresAt time.Time) bool {
	if expiresAt.IsZero() {
		return false
	}
	return time.Now().After(expiresAt)
}

// Config holds configuration for the AutoRefresher.
type Config struct {
	// RefreshThreshold is the duration before expiration when a token
	// should be proactively refreshed.
	RefreshThreshold time.Duration

	// CacheDuration is how long credentials are cached before being
	// re-read from the source.
	CacheDuration time.Duration

	// RateLimitInterval is the minimum time between refresh attempts.
	RateLimitInterval time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		RefreshThreshold:  10 * time.Minute,
		CacheDuration:     5 * time.Minute,
		RateLimitInterval: 30 * time.Second,
	}
}

// cachedCredentials holds credentials with cache metadata.
type cachedCredentials struct {
	creds    *Credentials
	cachedAt time.Time
}

// AutoRefresher manages automatic credential refresh with caching
// and rate limiting.
type AutoRefresher struct {
	mu        sync.Mutex
	reader    CredentialReader
	refresher TokenRefresher
	config    *Config
	endpoints map[string]string
	cache     map[string]*cachedCredentials
	lastRefr  map[string]time.Time
}

// NewAutoRefresher creates a new AutoRefresher.
// The endpoints map maps provider names to their token endpoint URLs.
func NewAutoRefresher(
	reader CredentialReader,
	refresher TokenRefresher,
	config *Config,
	endpoints map[string]string,
) *AutoRefresher {
	if config == nil {
		config = DefaultConfig()
	}
	return &AutoRefresher{
		reader:    reader,
		refresher: refresher,
		config:    config,
		endpoints: endpoints,
		cache:     make(map[string]*cachedCredentials),
		lastRefr:  make(map[string]time.Time),
	}
}

// GetCredentials returns valid credentials for the named provider,
// refreshing automatically if needed. It checks the cache first,
// reads from the credential source if the cache is stale, and
// refreshes the token if it is expiring or expired.
func (a *AutoRefresher) GetCredentials(
	providerName string,
) (*Credentials, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check cache
	if cached, ok := a.cache[providerName]; ok {
		age := time.Since(cached.cachedAt)
		if age < a.config.CacheDuration &&
			!cached.creds.NeedsRefresh(a.config.RefreshThreshold) {
			return cached.creds, nil
		}
	}

	// Read fresh credentials
	creds, err := a.reader.ReadCredentials(providerName)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read credentials for %s: %w", providerName, err,
		)
	}

	// Check if refresh is needed
	if creds.NeedsRefresh(a.config.RefreshThreshold) {
		refreshed, err := a.tryRefresh(providerName, creds)
		if err != nil {
			// If token is expired and refresh failed, return error
			if creds.IsExpired() {
				return nil, fmt.Errorf(
					"credentials for %s expired and refresh failed: %w",
					providerName, err,
				)
			}
			// Token still valid, use existing
		} else {
			creds = refreshed
		}
	}

	// Update cache
	a.cache[providerName] = &cachedCredentials{
		creds:    creds,
		cachedAt: time.Now(),
	}

	return creds, nil
}

// tryRefresh attempts to refresh credentials, respecting rate limits.
func (a *AutoRefresher) tryRefresh(
	providerName string, creds *Credentials,
) (*Credentials, error) {
	// Check rate limit
	if lastRefresh, ok := a.lastRefr[providerName]; ok {
		if time.Since(lastRefresh) < a.config.RateLimitInterval {
			return nil, fmt.Errorf(
				"refresh rate limited for %s: last attempt %v ago",
				providerName, time.Since(lastRefresh),
			)
		}
	}

	endpoint, ok := a.endpoints[providerName]
	if !ok {
		return nil, fmt.Errorf(
			"no token endpoint configured for provider: %s",
			providerName,
		)
	}

	if creds.RefreshToken == "" {
		return nil, fmt.Errorf(
			"no refresh token available for %s", providerName,
		)
	}

	a.lastRefr[providerName] = time.Now()

	refreshed, err := a.refresher.Refresh(creds.RefreshToken, endpoint)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to refresh token for %s: %w", providerName, err,
		)
	}

	// Carry over scopes and metadata if not returned
	if len(refreshed.Scopes) == 0 && len(creds.Scopes) > 0 {
		refreshed.Scopes = creds.Scopes
	}
	if len(refreshed.Metadata) == 0 && len(creds.Metadata) > 0 {
		refreshed.Metadata = creds.Metadata
	}

	return refreshed, nil
}

// ClearCache removes all cached credentials.
func (a *AutoRefresher) ClearCache() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache = make(map[string]*cachedCredentials)
}

// ClearCacheFor removes cached credentials for a specific provider.
func (a *AutoRefresher) ClearCacheFor(providerName string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.cache, providerName)
}
