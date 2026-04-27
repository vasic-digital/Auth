package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.auth/pkg/apikey"
	"digital.vasic.auth/pkg/jwt"
	"digital.vasic.auth/pkg/middleware"
	"digital.vasic.auth/pkg/oauth"
	"digital.vasic.auth/pkg/token"
)

func TestJWTCreateValidateRefreshFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	cfg := &jwt.Config{
		SigningMethod: nil,
		Secret:        []byte("integration-test-secret-key-32bytes!"),
		Expiration:    10 * time.Minute,
		Issuer:        "helix-integration",
	}
	cfg.SigningMethod = jwt.DefaultConfig("x").SigningMethod

	mgr := jwt.NewManager(cfg)

	claims := map[string]interface{}{
		"sub":    "user-123",
		"role":   "admin",
		"scopes": []string{"read", "write"},
	}

	tokenStr, err := mgr.Create(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	parsed, err := mgr.Validate(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "user-123", parsed.Claims["sub"])
	assert.Equal(t, "helix-integration", parsed.Claims["iss"])
	assert.False(t, parsed.ExpiresAt.IsZero())
	assert.False(t, parsed.IssuedAt.IsZero())
	assert.True(t, parsed.ExpiresAt.After(time.Now()))

	refreshed, err := mgr.Refresh(tokenStr)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed)
	// Note: don't assert inequality between tokenStr and refreshed —
	// JWT refresh is deterministic at second granularity, so fast
	// refreshes within the same second produce identical tokens.

	parsedRefreshed, err := mgr.Validate(refreshed)
	require.NoError(t, err)
	assert.Equal(t, "user-123", parsedRefreshed.Claims["sub"])
	// ExpiresAt must be >= parsed.ExpiresAt (equal when refresh happens
	// within the same second — the refreshed token is identical).
	assert.False(t, parsedRefreshed.ExpiresAt.Before(parsed.ExpiresAt),
		"refreshed expiry must not precede original")
}

func TestAPIKeyGenerateStoreValidateFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	gen := apikey.NewGenerator(&apikey.GeneratorConfig{
		Prefix: "hx-",
		Length: 24,
	})
	store := apikey.NewInMemoryStore()

	key1, err := gen.Generate("test-key-1", []string{"read", "write"}, time.Time{})
	require.NoError(t, err)

	key2, err := gen.Generate("test-key-2", []string{"read"}, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	require.NoError(t, store.Store(key1))
	require.NoError(t, store.Store(key2))

	validated, err := apikey.Validate(store, key1.Key)
	require.NoError(t, err)
	assert.Equal(t, "test-key-1", validated.Name)
	assert.True(t, validated.HasScope("read"))
	assert.True(t, validated.HasScope("write"))
	assert.True(t, validated.HasAllScopes([]string{"read", "write"}))
	assert.False(t, validated.HasScope("admin"))

	retrieved, err := store.GetByID(key1.ID)
	require.NoError(t, err)
	assert.Equal(t, key1.Key, retrieved.Key)

	require.NoError(t, store.Delete(key1.Key))

	_, err = store.Get(key1.Key)
	assert.Error(t, err)

	keys, err := store.List()
	require.NoError(t, err)
	assert.Len(t, keys, 1)
}

func TestMiddlewareBearerTokenIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	jwtMgr := jwt.NewManager(jwt.DefaultConfig("middleware-test-secret"))

	tokenStr, err := jwtMgr.Create(map[string]interface{}{
		"sub":  "user-42",
		"role": "editor",
	})
	require.NoError(t, err)

	validator := &jwtTokenValidator{mgr: jwtMgr}
	mw := middleware.BearerToken(validator)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.ClaimsFromContext(r.Context())
		assert.Equal(t, "user-42", claims["sub"])
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/resource", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	req2 := httptest.NewRequest("GET", "/api/resource", nil)
	req2.Header.Set("Authorization", "Bearer invalid-token")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
}

type jwtTokenValidator struct {
	mgr *jwt.Manager
}

func (v *jwtTokenValidator) ValidateToken(tokenStr string) (map[string]interface{}, error) {
	tok, err := v.mgr.Validate(tokenStr)
	if err != nil {
		return nil, err
	}
	return tok.Claims, nil
}

func TestOAuthFileCredentialReaderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "provider-creds.json")

	creds := oauth.Credentials{
		AccessToken:  "test-access-token-abc123",
		RefreshToken: "test-refresh-token-xyz789",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Scopes:       []string{"openid", "profile"},
	}
	data, err := json.Marshal(creds)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(credFile, data, 0600))

	reader := oauth.NewFileCredentialReader(map[string]string{
		"test-provider": credFile,
	})

	result, err := reader.ReadCredentials("test-provider")
	require.NoError(t, err)
	assert.Equal(t, "test-access-token-abc123", result.AccessToken)
	assert.Equal(t, "test-refresh-token-xyz789", result.RefreshToken)
	assert.False(t, result.IsExpired())
	assert.False(t, result.NeedsRefresh(5*time.Minute))

	_, err = reader.ReadCredentials("nonexistent-provider")
	assert.Error(t, err)
}

func TestTokenStoreWithTTLIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	store := token.NewInMemoryStore()

	tok := token.NewSimpleToken("access-1", "refresh-1", time.Now().Add(1*time.Hour))

	require.NoError(t, store.Set("session-1", tok, 0))

	retrieved, err := store.Get("session-1")
	require.NoError(t, err)
	assert.Equal(t, "access-1", retrieved.AccessToken())
	assert.Equal(t, "refresh-1", retrieved.RefreshToken())
	assert.False(t, retrieved.IsExpired())

	require.NoError(t, store.Revoke("session-1"))

	_, err = store.Get("session-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")

	removed := store.Cleanup()
	assert.Equal(t, 1, removed)
	assert.Equal(t, 0, store.Len())
}

func TestMiddlewareChainIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	keyValidator := &testAPIKeyValidator{
		keys: map[string][]string{
			"valid-key-123": {"read", "write", "admin"},
		},
	}

	chain := middleware.Chain(
		middleware.APIKeyHeader(keyValidator, "X-API-Key"),
		middleware.RequireScopes("read", "write"),
	)

	handler := chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := middleware.APIKeyFromContext(r.Context())
		scopes := middleware.ScopesFromContext(r.Context())
		assert.Equal(t, "valid-key-123", key)
		assert.Contains(t, scopes, "read")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "valid-key-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	req2 := httptest.NewRequest("GET", "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
}

type testAPIKeyValidator struct {
	keys map[string][]string
}

func (v *testAPIKeyValidator) ValidateKey(key string) ([]string, error) {
	scopes, ok := v.keys[key]
	if !ok {
		return nil, assert.AnError
	}
	return scopes, nil
}

func TestAutoRefresherIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.RefreshResponse{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "expiring-creds.json")
	expiring := oauth.Credentials{
		AccessToken:  "old-access-token",
		RefreshToken: "refresh-token-for-renewal",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
		Scopes:       []string{"api"},
	}
	data, _ := json.Marshal(expiring)
	os.WriteFile(credFile, data, 0600)

	reader := oauth.NewFileCredentialReader(map[string]string{
		"test": credFile,
	})
	refresher := oauth.NewHTTPTokenRefresher(nil, "client-id", nil)

	ar := oauth.NewAutoRefresher(reader, refresher, &oauth.Config{
		RefreshThreshold:  10 * time.Minute,
		CacheDuration:     5 * time.Minute,
		RateLimitInterval: 1 * time.Second,
	}, map[string]string{
		"test": srv.URL,
	})

	creds, err := ar.GetCredentials("test")
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", creds.AccessToken)

	_ = context.Background()
}
