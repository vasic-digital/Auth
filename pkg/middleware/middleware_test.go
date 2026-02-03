package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTokenValidator is a test helper implementing TokenValidator.
type testTokenValidator struct {
	claims map[string]interface{}
	err    error
}

func (v *testTokenValidator) ValidateToken(
	token string,
) (map[string]interface{}, error) {
	if v.err != nil {
		return nil, v.err
	}
	return v.claims, nil
}

// testAPIKeyValidator is a test helper implementing APIKeyValidator.
type testAPIKeyValidator struct {
	scopes []string
	err    error
}

func (v *testAPIKeyValidator) ValidateKey(
	key string,
) ([]string, error) {
	if v.err != nil {
		return nil, v.err
	}
	return v.scopes, nil
}

// okHandler is a simple handler that returns 200 OK.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestBearerToken_Valid(t *testing.T) {
	validator := &testTokenValidator{
		claims: map[string]interface{}{
			"sub":    "user-1",
			"scopes": []string{"read", "write"},
		},
	}

	handler := BearerToken(validator)(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBearerToken_MissingHeader(t *testing.T) {
	validator := &testTokenValidator{}
	handler := BearerToken(validator)(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing authorization")
}

func TestBearerToken_InvalidScheme(t *testing.T) {
	validator := &testTokenValidator{}
	handler := BearerToken(validator)(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid authorization scheme")
}

func TestBearerToken_EmptyToken(t *testing.T) {
	validator := &testTokenValidator{}
	handler := BearerToken(validator)(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "empty bearer token")
}

func TestBearerToken_InvalidToken(t *testing.T) {
	validator := &testTokenValidator{
		err: fmt.Errorf("invalid token"),
	}
	handler := BearerToken(validator)(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid token")
}

func TestBearerToken_ClaimsInContext(t *testing.T) {
	validator := &testTokenValidator{
		claims: map[string]interface{}{
			"sub": "user-42",
		},
	}

	var capturedClaims map[string]interface{}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClaims = ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := BearerToken(validator)(inner)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.NotNil(t, capturedClaims)
	assert.Equal(t, "user-42", capturedClaims["sub"])
}

func TestAPIKeyHeader_Valid(t *testing.T) {
	validator := &testAPIKeyValidator{
		scopes: []string{"read", "write"},
	}

	handler := APIKeyHeader(validator, "X-API-Key")(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "ak-test123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIKeyHeader_MissingHeader(t *testing.T) {
	validator := &testAPIKeyValidator{}
	handler := APIKeyHeader(validator, "X-API-Key")(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing API key")
}

func TestAPIKeyHeader_InvalidKey(t *testing.T) {
	validator := &testAPIKeyValidator{
		err: fmt.Errorf("invalid key"),
	}
	handler := APIKeyHeader(validator, "X-API-Key")(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "bad-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid API key")
}

func TestAPIKeyHeader_ScopesAndKeyInContext(t *testing.T) {
	validator := &testAPIKeyValidator{
		scopes: []string{"admin"},
	}

	var capturedScopes []string
	var capturedKey string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedScopes = ScopesFromContext(r.Context())
		capturedKey = APIKeyFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := APIKeyHeader(validator, "X-API-Key")(inner)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "ak-mykey")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, []string{"admin"}, capturedScopes)
	assert.Equal(t, "ak-mykey", capturedKey)
}

func TestRequireScopes_HasAll(t *testing.T) {
	inner := okHandler()
	handler := RequireScopes("read", "write")(inner)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(
		req.Context(), ScopesKey, []string{"read", "write", "admin"},
	)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireScopes_MissingScope(t *testing.T) {
	handler := RequireScopes("read", "admin")(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(
		req.Context(), ScopesKey, []string{"read"},
	)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "insufficient scopes")
}

func TestRequireScopes_NoScopes(t *testing.T) {
	handler := RequireScopes("read")(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "no scopes in context")
}

func TestChain(t *testing.T) {
	validator := &testTokenValidator{
		claims: map[string]interface{}{
			"sub":    "user-1",
			"scopes": []string{"read", "write"},
		},
	}

	chained := Chain(
		BearerToken(validator),
		RequireScopes("read"),
	)

	handler := chained(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestChain_FailsAtFirstMiddleware(t *testing.T) {
	validator := &testTokenValidator{
		err: fmt.Errorf("invalid"),
	}

	chained := Chain(
		BearerToken(validator),
		RequireScopes("admin"),
	)

	handler := chained(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestChain_Empty(t *testing.T) {
	chained := Chain()
	handler := chained(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestClaimsFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	claims := ClaimsFromContext(ctx)
	assert.Nil(t, claims)
}

func TestScopesFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	scopes := ScopesFromContext(ctx)
	assert.Nil(t, scopes)
}

func TestAPIKeyFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	key := APIKeyFromContext(ctx)
	assert.Empty(t, key)
}

func TestRequireScopes_InvalidScopesType(t *testing.T) {
	// Test when scopes in context is not []string (line 145-152)
	handler := RequireScopes("read")(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	// Set scopes as wrong type (string instead of []string)
	ctx := context.WithValue(req.Context(), ScopesKey, "not-a-slice")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid scopes in context")
}

func TestRequireScopes_InvalidScopesType_Int(t *testing.T) {
	// Test when scopes in context is an int
	handler := RequireScopes("admin")(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ScopesKey, 12345)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid scopes in context")
}

func TestRequireScopes_InvalidScopesType_Map(t *testing.T) {
	// Test when scopes in context is a map
	handler := RequireScopes("write")(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ScopesKey, map[string]bool{"read": true})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid scopes in context")
}

func TestBearerToken_ScopesExtraction_NonSliceType(t *testing.T) {
	// Test when claims["scopes"] exists but is not []string
	// This exercises lines 86-90 where scopes extraction may fail
	validator := &testTokenValidator{
		claims: map[string]interface{}{
			"sub":    "user-1",
			"scopes": "read,write", // String instead of []string
		},
	}

	var capturedScopes []string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedScopes = ScopesFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := BearerToken(validator)(inner)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Scopes should be nil because it wasn't a []string
	assert.Nil(t, capturedScopes)
}

func TestBearerToken_ScopesExtraction_ValidSlice(t *testing.T) {
	// Test when claims["scopes"] is a valid []string
	validator := &testTokenValidator{
		claims: map[string]interface{}{
			"sub":    "user-1",
			"scopes": []string{"read", "write", "admin"},
		},
	}

	var capturedScopes []string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedScopes = ScopesFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := BearerToken(validator)(inner)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []string{"read", "write", "admin"}, capturedScopes)
}
