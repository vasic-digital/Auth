package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentials_IsExpired(t *testing.T) {
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
			c := &Credentials{ExpiresAt: tt.expiresAt}
			assert.Equal(t, tt.expected, c.IsExpired())
		})
	}
}

func TestCredentials_NeedsRefresh(t *testing.T) {
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
			name:      "zero time",
			expiresAt: time.Time{},
			threshold: 10 * time.Minute,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Credentials{ExpiresAt: tt.expiresAt}
			assert.Equal(t, tt.expected, c.NeedsRefresh(tt.threshold))
		})
	}
}

func TestNeedsRefresh(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		threshold time.Duration
		expected  bool
	}{
		{
			name:      "needs refresh",
			expiresAt: time.Now().Add(5 * time.Minute),
			threshold: 10 * time.Minute,
			expected:  true,
		},
		{
			name:      "does not need refresh",
			expiresAt: time.Now().Add(30 * time.Minute),
			threshold: 10 * time.Minute,
			expected:  false,
		},
		{
			name:      "zero time",
			expiresAt: time.Time{},
			threshold: 10 * time.Minute,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NeedsRefresh(tt.expiresAt, tt.threshold))
		})
	}
}

func TestIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "expired",
			expiresAt: time.Now().Add(-time.Hour),
			expected:  true,
		},
		{
			name:      "not expired",
			expiresAt: time.Now().Add(time.Hour),
			expected:  false,
		},
		{
			name:      "zero time",
			expiresAt: time.Time{},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsExpired(tt.expiresAt))
		})
	}
}

func TestFileCredentialReader_ReadCredentials(t *testing.T) {
	// Create temp credential files
	tmpDir := t.TempDir()

	validCreds := Credentials{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		Scopes:       []string{"read", "write"},
	}
	validData, err := json.Marshal(validCreds)
	require.NoError(t, err)

	validPath := filepath.Join(tmpDir, "valid.json")
	require.NoError(t, os.WriteFile(validPath, validData, 0600))

	emptyTokenPath := filepath.Join(tmpDir, "empty.json")
	emptyTokenData, _ := json.Marshal(
		Credentials{AccessToken: ""},
	)
	require.NoError(t, os.WriteFile(
		emptyTokenPath, emptyTokenData, 0600,
	))

	invalidJSONPath := filepath.Join(tmpDir, "invalid.json")
	require.NoError(t, os.WriteFile(
		invalidJSONPath, []byte("{bad json"), 0600,
	))

	tests := []struct {
		name       string
		paths      map[string]string
		provider   string
		wantErr    bool
		errContain string
	}{
		{
			name:     "valid credentials",
			paths:    map[string]string{"provider1": validPath},
			provider: "provider1",
			wantErr:  false,
		},
		{
			name:       "unknown provider",
			paths:      map[string]string{},
			provider:   "unknown",
			wantErr:    true,
			errContain: "no credential path configured",
		},
		{
			name:       "file not found",
			paths:      map[string]string{"p": "/nonexistent/path"},
			provider:   "p",
			wantErr:    true,
			errContain: "not found",
		},
		{
			name:       "empty access token",
			paths:      map[string]string{"p": emptyTokenPath},
			provider:   "p",
			wantErr:    true,
			errContain: "empty access token",
		},
		{
			name:       "invalid JSON",
			paths:      map[string]string{"p": invalidJSONPath},
			provider:   "p",
			wantErr:    true,
			errContain: "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewFileCredentialReader(tt.paths)
			creds, err := reader.ReadCredentials(tt.provider)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "test-access-token", creds.AccessToken)
				assert.Equal(t, "test-refresh-token", creds.RefreshToken)
			}
		})
	}
}

func TestHTTPTokenRefresher_Refresh(t *testing.T) {
	tests := []struct {
		name         string
		refreshToken string
		serverStatus int
		serverResp   interface{}
		wantErr      bool
		errContain   string
	}{
		{
			name:         "successful refresh",
			refreshToken: "valid-refresh-token",
			serverStatus: http.StatusOK,
			serverResp: RefreshResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
				TokenType:    "Bearer",
				Scope:        "read write",
			},
			wantErr: false,
		},
		{
			name:         "empty refresh token",
			refreshToken: "",
			wantErr:      true,
			errContain:   "no refresh token",
		},
		{
			name:         "server error",
			refreshToken: "valid-token",
			serverStatus: http.StatusBadRequest,
			serverResp:   map[string]string{"error": "invalid_grant"},
			wantErr:      true,
			errContain:   "refresh failed with status 400",
		},
		{
			name:         "keeps existing refresh token",
			refreshToken: "original-refresh",
			serverStatus: http.StatusOK,
			serverResp: RefreshResponse{
				AccessToken: "new-access",
				ExpiresIn:   3600,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			if tt.serverStatus > 0 {
				server = httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "POST", r.Method)
						assert.Equal(t,
							"application/x-www-form-urlencoded",
							r.Header.Get("Content-Type"),
						)

						w.WriteHeader(tt.serverStatus)
						respData, _ := json.Marshal(tt.serverResp)
						_, _ = w.Write(respData)
					}),
				)
				defer server.Close()
			}

			refresher := NewHTTPTokenRefresher(nil, "test-client", nil)

			endpoint := ""
			if server != nil {
				endpoint = server.URL
			}

			creds, err := refresher.Refresh(tt.refreshToken, endpoint)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, creds.AccessToken)
				assert.NotEmpty(t, creds.RefreshToken)
				assert.False(t, creds.ExpiresAt.IsZero())
			}
		})
	}
}

func TestHTTPTokenRefresher_Refresh_WithExtraParams(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseForm()
			require.NoError(t, err)
			assert.Equal(t, "custom-value", r.FormValue("custom_param"))
			assert.Equal(t, "test-client", r.FormValue("client_id"))

			resp := RefreshResponse{
				AccessToken: "new-token",
				ExpiresIn:   3600,
			}
			respData, _ := json.Marshal(resp)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(respData)
		}),
	)
	defer server.Close()

	refresher := NewHTTPTokenRefresher(
		nil,
		"test-client",
		map[string]string{"custom_param": "custom-value"},
	)

	creds, err := refresher.Refresh("refresh-token", server.URL)
	require.NoError(t, err)
	assert.Equal(t, "new-token", creds.AccessToken)
}

// mockReader is a test helper implementing CredentialReader.
type mockReader struct {
	creds map[string]*Credentials
	err   error
}

func (m *mockReader) ReadCredentials(
	name string,
) (*Credentials, error) {
	if m.err != nil {
		return nil, m.err
	}
	c, ok := m.creds[name]
	if !ok {
		return nil, fmt.Errorf("not found: %s", name)
	}
	return c, nil
}

// mockRefresher is a test helper implementing TokenRefresher.
type mockRefresher struct {
	creds *Credentials
	err   error
}

func (m *mockRefresher) Refresh(
	refreshToken, endpoint string,
) (*Credentials, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.creds, nil
}

func TestAutoRefresher_GetCredentials_Cached(t *testing.T) {
	creds := &Credentials{
		AccessToken:  "cached-token",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	reader := &mockReader{
		creds: map[string]*Credentials{"provider": creds},
	}

	ar := NewAutoRefresher(reader, nil, nil, nil)

	// First call reads from reader
	got, err := ar.GetCredentials("provider")
	require.NoError(t, err)
	assert.Equal(t, "cached-token", got.AccessToken)

	// Second call should use cache
	got, err = ar.GetCredentials("provider")
	require.NoError(t, err)
	assert.Equal(t, "cached-token", got.AccessToken)
}

func TestAutoRefresher_GetCredentials_Refresh(t *testing.T) {
	expiringSoon := &Credentials{
		AccessToken:  "old-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	reader := &mockReader{
		creds: map[string]*Credentials{"provider": expiringSoon},
	}

	refreshedCreds := &Credentials{
		AccessToken:  "new-token",
		RefreshToken: "new-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	refresher := &mockRefresher{creds: refreshedCreds}

	cfg := DefaultConfig()
	cfg.RefreshThreshold = 10 * time.Minute

	ar := NewAutoRefresher(
		reader, refresher, cfg,
		map[string]string{"provider": "http://example.com/token"},
	)

	got, err := ar.GetCredentials("provider")
	require.NoError(t, err)
	assert.Equal(t, "new-token", got.AccessToken)
}

func TestAutoRefresher_GetCredentials_RefreshRateLimited(t *testing.T) {
	expiringSoon := &Credentials{
		AccessToken:  "old-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	reader := &mockReader{
		creds: map[string]*Credentials{"provider": expiringSoon},
	}

	refreshedCreds := &Credentials{
		AccessToken:  "new-token",
		RefreshToken: "new-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	refresher := &mockRefresher{creds: refreshedCreds}

	cfg := DefaultConfig()
	cfg.RefreshThreshold = 10 * time.Minute
	cfg.RateLimitInterval = time.Hour // Very long rate limit

	ar := NewAutoRefresher(
		reader, refresher, cfg,
		map[string]string{"provider": "http://example.com/token"},
	)

	// First call succeeds with refresh
	got, err := ar.GetCredentials("provider")
	require.NoError(t, err)
	assert.Equal(t, "new-token", got.AccessToken)

	// Clear cache to force re-read
	ar.ClearCache()

	// Second call should be rate limited, uses existing token
	got, err = ar.GetCredentials("provider")
	require.NoError(t, err)
	// Should use old token since refresh is rate limited and token
	// is not yet expired
	assert.Equal(t, "old-token", got.AccessToken)
}

func TestAutoRefresher_GetCredentials_ExpiredNoRefresh(t *testing.T) {
	expired := &Credentials{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().Add(-time.Hour),
	}

	reader := &mockReader{
		creds: map[string]*Credentials{"provider": expired},
	}

	cfg := DefaultConfig()
	cfg.RefreshThreshold = 10 * time.Minute

	ar := NewAutoRefresher(reader, nil, cfg, nil)

	_, err := ar.GetCredentials("provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestAutoRefresher_ClearCache(t *testing.T) {
	creds := &Credentials{
		AccessToken: "token",
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	reader := &mockReader{
		creds: map[string]*Credentials{"p": creds},
	}

	ar := NewAutoRefresher(reader, nil, nil, nil)

	_, err := ar.GetCredentials("p")
	require.NoError(t, err)

	ar.ClearCache()

	// Should re-read after cache clear
	_, err = ar.GetCredentials("p")
	require.NoError(t, err)
}

func TestAutoRefresher_ClearCacheFor(t *testing.T) {
	creds1 := &Credentials{
		AccessToken: "token1",
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	creds2 := &Credentials{
		AccessToken: "token2",
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	reader := &mockReader{
		creds: map[string]*Credentials{
			"p1": creds1,
			"p2": creds2,
		},
	}

	ar := NewAutoRefresher(reader, nil, nil, nil)

	_, err := ar.GetCredentials("p1")
	require.NoError(t, err)
	_, err = ar.GetCredentials("p2")
	require.NoError(t, err)

	ar.ClearCacheFor("p1")

	// p1 cache should be cleared, p2 should remain
	assert.Nil(t, ar.cache["p1"])
	assert.NotNil(t, ar.cache["p2"])
}

func TestDefaultConfig_Values(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 10*time.Minute, cfg.RefreshThreshold)
	assert.Equal(t, 5*time.Minute, cfg.CacheDuration)
	assert.Equal(t, 30*time.Second, cfg.RateLimitInterval)
}

func TestFileCredentialReader_ReadCredentials_ReadError(t *testing.T) {
	// Test non-existent read errors (permission denied)
	// Create a directory with no read permission to simulate read error
	tmpDir := t.TempDir()
	restrictedDir := filepath.Join(tmpDir, "restricted")
	require.NoError(t, os.Mkdir(restrictedDir, 0000))
	defer func() { _ = os.Chmod(restrictedDir, 0755) }()

	restrictedPath := filepath.Join(restrictedDir, "creds.json")

	reader := NewFileCredentialReader(map[string]string{
		"provider": restrictedPath,
	})

	_, err := reader.ReadCredentials("provider")
	assert.Error(t, err)
	// On Linux, attempting to read a file in a dir with no permissions
	// results in "permission denied" or similar error
	assert.Contains(t, err.Error(), "failed to read")
}

func TestHTTPTokenRefresher_Refresh_InvalidEndpoint(t *testing.T) {
	// Test with an invalid URL that causes http.NewRequest to fail
	refresher := NewHTTPTokenRefresher(nil, "", nil)

	// Using control characters in URL makes NewRequest fail
	_, err := refresher.Refresh("refresh-token", "http://[::1]:namedport")
	assert.Error(t, err)
	// Note: Different Go versions may produce different error messages
}

func TestHTTPTokenRefresher_Refresh_NetworkError(t *testing.T) {
	// Test with unreachable endpoint to trigger client.Do error
	client := &http.Client{Timeout: 100 * time.Millisecond}
	refresher := NewHTTPTokenRefresher(client, "", nil)

	// Use a non-routable address that will timeout/fail
	_, err := refresher.Refresh(
		"refresh-token",
		"http://192.0.2.1:12345/token", // TEST-NET-1, non-routable
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refresh request failed")
}

func TestHTTPTokenRefresher_Refresh_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{invalid json"))
		}),
	)
	defer server.Close()

	refresher := NewHTTPTokenRefresher(nil, "", nil)
	_, err := refresher.Refresh("refresh-token", server.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse refresh response")
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func (e *errorReader) Close() error {
	return nil
}

func TestHTTPTokenRefresher_Refresh_BodyReadError(t *testing.T) {
	// Create a custom transport that returns a response with an error reader
	transport := &errorTransport{}

	client := &http.Client{Transport: transport}
	refresher := NewHTTPTokenRefresher(client, "", nil)

	_, err := refresher.Refresh("refresh-token", "http://example.com/token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read refresh response")
}

// errorTransport is a custom RoundTripper that returns a response
// with an error-producing body reader.
type errorTransport struct{}

func (t *errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       &errorReader{},
		Header:     make(http.Header),
	}, nil
}

func TestAutoRefresher_GetCredentials_ReaderError(t *testing.T) {
	reader := &mockReader{
		err: fmt.Errorf("read error"),
	}

	ar := NewAutoRefresher(reader, nil, nil, nil)

	_, err := ar.GetCredentials("provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read credentials")
}

func TestAutoRefresher_tryRefresh_NoEndpoint(t *testing.T) {
	expiringSoon := &Credentials{
		AccessToken:  "old-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(-time.Hour), // Already expired
	}

	reader := &mockReader{
		creds: map[string]*Credentials{"provider": expiringSoon},
	}

	cfg := DefaultConfig()
	cfg.RefreshThreshold = 10 * time.Minute

	// No endpoints configured
	ar := NewAutoRefresher(reader, nil, cfg, nil)

	_, err := ar.GetCredentials("provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no token endpoint")
}

func TestAutoRefresher_tryRefresh_NoRefreshToken(t *testing.T) {
	expiringSoon := &Credentials{
		AccessToken: "old-token",
		// No RefreshToken
		ExpiresAt: time.Now().Add(-time.Hour), // Already expired
	}

	reader := &mockReader{
		creds: map[string]*Credentials{"provider": expiringSoon},
	}

	cfg := DefaultConfig()
	cfg.RefreshThreshold = 10 * time.Minute

	ar := NewAutoRefresher(
		reader, nil, cfg,
		map[string]string{"provider": "http://example.com/token"},
	)

	_, err := ar.GetCredentials("provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no refresh token")
}

func TestAutoRefresher_tryRefresh_CarriesOverScopesAndMetadata(t *testing.T) {
	originalCreds := &Credentials{
		AccessToken:  "old-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
		Scopes:       []string{"read", "write"},
		Metadata: map[string]interface{}{
			"custom_key": "custom_value",
		},
	}

	reader := &mockReader{
		creds: map[string]*Credentials{"provider": originalCreds},
	}

	// Refreshed creds without scopes/metadata
	refreshedCreds := &Credentials{
		AccessToken:  "new-token",
		RefreshToken: "new-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
		// No Scopes or Metadata returned
	}

	refresher := &mockRefresher{creds: refreshedCreds}

	cfg := DefaultConfig()
	cfg.RefreshThreshold = 10 * time.Minute

	ar := NewAutoRefresher(
		reader, refresher, cfg,
		map[string]string{"provider": "http://example.com/token"},
	)

	got, err := ar.GetCredentials("provider")
	require.NoError(t, err)
	assert.Equal(t, "new-token", got.AccessToken)
	// Scopes and metadata should be carried over
	assert.Equal(t, []string{"read", "write"}, got.Scopes)
	assert.Equal(t, "custom_value", got.Metadata["custom_key"])
}

func TestAutoRefresher_tryRefresh_RefresherError(t *testing.T) {
	expiringSoon := &Credentials{
		AccessToken:  "old-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(-time.Hour), // Already expired
	}

	reader := &mockReader{
		creds: map[string]*Credentials{"provider": expiringSoon},
	}

	refresher := &mockRefresher{
		err: fmt.Errorf("refresh failed"),
	}

	cfg := DefaultConfig()
	cfg.RefreshThreshold = 10 * time.Minute

	ar := NewAutoRefresher(
		reader, refresher, cfg,
		map[string]string{"provider": "http://example.com/token"},
	)

	_, err := ar.GetCredentials("provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refresh failed")
}

func TestHTTPTokenRefresher_Refresh_NoExpiresIn(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := RefreshResponse{
				AccessToken: "new-token",
				// No ExpiresIn
			}
			respData, _ := json.Marshal(resp)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(respData)
		}),
	)
	defer server.Close()

	refresher := NewHTTPTokenRefresher(nil, "", nil)
	creds, err := refresher.Refresh("refresh-token", server.URL)
	require.NoError(t, err)
	assert.Equal(t, "new-token", creds.AccessToken)
	// ExpiresAt should be zero since no ExpiresIn was returned
	assert.True(t, creds.ExpiresAt.IsZero())
}

func TestHTTPTokenRefresher_Refresh_NoClientID(t *testing.T) {
	var receivedClientID string

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseForm()
			require.NoError(t, err)
			receivedClientID = r.FormValue("client_id")

			resp := RefreshResponse{
				AccessToken: "new-token",
				ExpiresIn:   3600,
			}
			respData, _ := json.Marshal(resp)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(respData)
		}),
	)
	defer server.Close()

	// Create refresher with empty clientID
	refresher := NewHTTPTokenRefresher(nil, "", nil)
	_, err := refresher.Refresh("refresh-token", server.URL)
	require.NoError(t, err)
	// client_id should not be sent
	assert.Empty(t, receivedClientID)
}
