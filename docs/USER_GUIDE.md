# Auth Module - User Guide

## Overview

`digital.vasic.auth` is a generic, reusable Go module for authentication and authorization. It provides five independent packages that can be used individually or composed together:

- **jwt** -- JWT token creation, validation, and refresh
- **apikey** -- API key generation, validation, and pluggable storage
- **oauth** -- OAuth2 credential reading, token refresh, and auto-refresh
- **middleware** -- HTTP authentication middleware for `net/http`
- **token** -- Token interface, Claims map, and in-memory store with TTL

## Installation

```bash
go get digital.vasic.auth
```

Requires Go 1.24 or later.

## JWT Authentication

The `jwt` package provides a `Manager` for creating, validating, and refreshing JWT tokens using configurable signing methods.

### Basic Setup

```go
import "digital.vasic.auth/pkg/jwt"

// Create a manager with default config (HS256, 1-hour expiration)
cfg := jwt.DefaultConfig("your-secret-key-at-least-32-bytes")
cfg.Issuer = "my-service"
manager := jwt.NewManager(cfg)
```

### Creating Tokens

```go
tokenStr, err := manager.Create(map[string]interface{}{
    "sub":  "user-123",
    "role": "admin",
    "scopes": []string{"read", "write"},
})
if err != nil {
    log.Fatalf("failed to create token: %v", err)
}
// tokenStr is a signed JWT string
```

The `Create` method automatically sets `iat` (issued at), `exp` (expiration), and `iss` (issuer, if configured). User-provided claims are merged in.

### Validating Tokens

```go
token, err := manager.Validate(tokenStr)
if err != nil {
    log.Fatalf("invalid token: %v", err)
}

fmt.Println(token.Claims["sub"])  // "user-123"
fmt.Println(token.Claims["role"]) // "admin"
fmt.Println(token.ExpiresAt)      // time.Time
fmt.Println(token.IssuedAt)       // time.Time
fmt.Println(token.Raw)            // original token string
```

Validation checks the signature, signing method, and expiration. An error is returned if any check fails.

### Refreshing Tokens

```go
newTokenStr, err := manager.Refresh(tokenStr)
if err != nil {
    log.Fatalf("cannot refresh: %v", err)
}
```

Refresh validates the existing token, extracts user claims (excluding `exp`, `iat`, `iss`), and creates a new token with fresh expiration.

### Custom Configuration

```go
cfg := &jwt.Config{
    SigningMethod: gojwt.SigningMethodHS384,
    Secret:        []byte("your-384-bit-secret"),
    Expiration:    24 * time.Hour,
    Issuer:        "auth-service",
}
manager := jwt.NewManager(cfg)
```

## API Key Authentication

The `apikey` package generates cryptographically random API keys with configurable prefixes and provides a pluggable storage backend.

### Generating Keys

```go
import "digital.vasic.auth/pkg/apikey"

gen := apikey.NewGenerator(&apikey.GeneratorConfig{
    Prefix: "sk-",
    Length: 32, // 32 random bytes = 64 hex chars
})

key, err := gen.Generate("production-key", []string{"read", "write"}, time.Time{})
if err != nil {
    log.Fatalf("failed to generate key: %v", err)
}

fmt.Println(key.ID)        // UUID
fmt.Println(key.Key)       // "sk-<64 hex chars>"
fmt.Println(key.Name)      // "production-key"
fmt.Println(key.Scopes)    // ["read", "write"]
fmt.Println(key.CreatedAt) // time.Time
```

Use `DefaultGeneratorConfig()` for defaults: prefix `"ak-"` and 32-byte length.

### Generating Keys with Expiration

```go
expiration := time.Now().Add(90 * 24 * time.Hour) // 90 days
key, err := gen.Generate("temp-key", []string{"read"}, expiration)
```

### Storing and Retrieving Keys

```go
store := apikey.NewInMemoryStore()

// Store a key
err := store.Store(key)

// Retrieve by key string
retrieved, err := store.Get(key.Key)

// Retrieve by UUID
retrieved, err := store.GetByID(key.ID)

// List all keys
allKeys, err := store.List()

// Delete a key
err = store.Delete(key.Key)
```

### Validating Keys

```go
validated, err := apikey.Validate(store, "sk-abc123...")
if err != nil {
    // Key not found or expired
    log.Printf("invalid key: %v", err)
}

// Check scopes
if validated.HasScope("write") {
    // Key has write permission
}

if validated.HasAllScopes([]string{"read", "write"}) {
    // Key has both permissions
}
```

### Masking Keys for Display

```go
masked := apikey.MaskKey("sk-abcdef1234567890abcdef1234567890")
// Output: "sk-****************************7890"
```

### Custom Storage Backend

Implement the `KeyStore` interface to use any storage backend:

```go
type KeyStore interface {
    Store(key *APIKey) error
    Get(keyString string) (*APIKey, error)
    GetByID(id string) (*APIKey, error)
    Delete(keyString string) error
    List() ([]*APIKey, error)
}
```

Example PostgreSQL implementation:

```go
type PostgresKeyStore struct {
    db *sql.DB
}

func (s *PostgresKeyStore) Store(key *apikey.APIKey) error {
    _, err := s.db.Exec(
        "INSERT INTO api_keys (id, key, name, scopes, expires_at, created_at) VALUES ($1,$2,$3,$4,$5,$6)",
        key.ID, key.Key, key.Name, pq.Array(key.Scopes), key.ExpiresAt, key.CreatedAt,
    )
    return err
}
// ... implement remaining methods
```

## OAuth2 Credential Management

The `oauth` package manages OAuth2 credentials with file-based reading, HTTP-based token refresh, and automatic refresh with caching and rate limiting.

### Reading Credentials from Files

```go
import "digital.vasic.auth/pkg/oauth"

reader := oauth.NewFileCredentialReader(map[string]string{
    "github": "/etc/auth/github-creds.json",
    "google": "/etc/auth/google-creds.json",
})

creds, err := reader.ReadCredentials("github")
if err != nil {
    log.Fatalf("failed to read credentials: %v", err)
}

fmt.Println(creds.AccessToken)
fmt.Println(creds.RefreshToken)
fmt.Println(creds.ExpiresAt)
fmt.Println(creds.Scopes)
```

The credential JSON file format:

```json
{
    "access_token": "gho_xxxxxxxxxxxx",
    "refresh_token": "ghr_xxxxxxxxxxxx",
    "expires_at": "2025-12-31T23:59:59Z",
    "scopes": ["repo", "user"],
    "metadata": {
        "org": "my-org"
    }
}
```

### Refreshing Tokens via HTTP

```go
refresher := oauth.NewHTTPTokenRefresher(
    nil,         // uses default http.Client with 30s timeout
    "client-id", // OAuth2 client ID
    map[string]string{
        "client_secret": "secret-value",
    },
)

newCreds, err := refresher.Refresh(
    creds.RefreshToken,
    "https://github.com/login/oauth/access_token",
)
```

The refresher sends a standard OAuth2 `refresh_token` grant POST request and parses the response.

### Auto-Refresh with Caching

The `AutoRefresher` combines credential reading and refresh into a single call that handles caching, expiration detection, and rate limiting:

```go
config := &oauth.Config{
    RefreshThreshold:  10 * time.Minute, // refresh 10 min before expiry
    CacheDuration:     5 * time.Minute,  // cache for 5 min
    RateLimitInterval: 30 * time.Second, // min 30s between refreshes
}

ar := oauth.NewAutoRefresher(
    reader,
    refresher,
    config, // nil for defaults
    map[string]string{
        "github": "https://github.com/login/oauth/access_token",
        "google": "https://oauth2.googleapis.com/token",
    },
)

// Always returns valid credentials, refreshing if needed
creds, err := ar.GetCredentials("github")
```

The `AutoRefresher`:
1. Checks the internal cache first
2. Reads fresh credentials from the `CredentialReader` if cache is stale
3. Proactively refreshes if credentials are within the threshold of expiration
4. Rate-limits refresh attempts to prevent flooding the token endpoint
5. Falls back to existing (still valid) credentials if refresh fails

### Cache Management

```go
ar.ClearCache()                // Clear all cached credentials
ar.ClearCacheFor("github")    // Clear cache for a specific provider
```

### Utility Functions

```go
// Check if a token needs refreshing
if oauth.NeedsRefresh(creds.ExpiresAt, 10*time.Minute) {
    // Token expires within 10 minutes
}

// Check if a token is expired
if oauth.IsExpired(creds.ExpiresAt) {
    // Token has expired
}
```

## HTTP Middleware

The `middleware` package provides standard `net/http` middleware for Bearer token and API key authentication.

### Bearer Token Authentication

```go
import "digital.vasic.auth/pkg/middleware"

// Implement TokenValidator (typically wraps jwt.Manager)
type jwtValidator struct {
    manager *jwt.Manager
}

func (v *jwtValidator) ValidateToken(token string) (map[string]interface{}, error) {
    t, err := v.manager.Validate(token)
    if err != nil {
        return nil, err
    }
    return t.Claims, nil
}

// Apply middleware
validator := &jwtValidator{manager: jwtManager}
handler := middleware.BearerToken(validator)(myHandler)

http.Handle("/api/protected", handler)
```

The middleware extracts the token from the `Authorization: Bearer <token>` header, validates it, and stores claims in the request context.

### API Key Header Authentication

```go
// Implement APIKeyValidator (typically wraps apikey store)
type keyValidator struct {
    store apikey.KeyStore
}

func (v *keyValidator) ValidateKey(key string) ([]string, error) {
    apiKey, err := apikey.Validate(v.store, key)
    if err != nil {
        return nil, err
    }
    return apiKey.Scopes, nil
}

// Apply middleware with custom header name
validator := &keyValidator{store: keyStore}
handler := middleware.APIKeyHeader(validator, "X-API-Key")(myHandler)
```

### Scope Validation

`RequireScopes` checks that the request context contains all required scopes. Place it after `BearerToken` or `APIKeyHeader`:

```go
handler := middleware.Chain(
    middleware.BearerToken(tokenValidator),
    middleware.RequireScopes("read", "write"),
)(myHandler)
```

If the required scopes are not present, a `403 Forbidden` response is returned.

### Chaining Middleware

Use `Chain` to compose multiple middleware in order:

```go
protectedHandler := middleware.Chain(
    middleware.BearerToken(tokenValidator),
    middleware.RequireScopes("admin"),
)(adminHandler)

apiHandler := middleware.Chain(
    middleware.APIKeyHeader(keyValidator, "X-API-Key"),
    middleware.RequireScopes("read"),
)(readHandler)
```

Middleware are applied in order -- the first in the list is the outermost wrapper.

### Extracting Context Values

After authentication middleware runs, use the accessor functions in downstream handlers:

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    // Get claims set by BearerToken middleware
    claims := middleware.ClaimsFromContext(r.Context())
    userID := claims["sub"].(string)

    // Get scopes set by BearerToken or APIKeyHeader middleware
    scopes := middleware.ScopesFromContext(r.Context())

    // Get API key set by APIKeyHeader middleware
    apiKey := middleware.APIKeyFromContext(r.Context())
}
```

## Token Store

The `token` package provides a generic `Token` interface, `Claims` helper type, and an in-memory store with TTL.

### Using SimpleToken

```go
import "digital.vasic.auth/pkg/token"

tk := token.NewSimpleToken(
    "access-token-string",
    "refresh-token-string",
    time.Now().Add(time.Hour),
)

fmt.Println(tk.AccessToken())                     // "access-token-string"
fmt.Println(tk.IsExpired())                        // false
fmt.Println(tk.NeedsRefresh(10 * time.Minute))    // false (if >10 min left)
```

### Using the In-Memory Store

```go
store := token.NewInMemoryStore()

// Store a token with 1-hour TTL in the store
err := store.Set("session-123", tk, time.Hour)

// Retrieve a token
retrieved, err := store.Get("session-123")

// Revoke a token (remains in store but Get returns error)
err = store.Revoke("session-123")

// Delete a token
err = store.Delete("session-123")

// Cleanup expired and revoked entries
removed := store.Cleanup()
fmt.Printf("removed %d entries\n", removed)

// Check store size
fmt.Printf("store has %d entries\n", store.Len())
```

### Working with Claims

```go
claims := token.Claims{
    "sub": "user-123",
    "iss": "my-service",
    "aud": "my-app",
    "exp": float64(time.Now().Add(time.Hour).Unix()),
    "iat": float64(time.Now().Unix()),
    "role": "admin",
}

fmt.Println(claims.Subject())            // "user-123"
fmt.Println(claims.Issuer())             // "my-service"
fmt.Println(claims.Audience())           // "my-app"
fmt.Println(claims.ExpiresAt())          // time.Time
fmt.Println(claims.IssuedAt())           // time.Time
fmt.Println(claims.Get("role"))          // "admin"
fmt.Println(claims.GetString("role"))    // "admin"
```

### Implementing Custom Token Types

Implement the `Token` interface for custom token types:

```go
type Token interface {
    AccessToken() string
    RefreshToken() string
    ExpiresAt() time.Time
    IsExpired() bool
    NeedsRefresh(threshold time.Duration) bool
}
```

Example with OAuth credentials:

```go
type OAuthToken struct {
    creds *oauth.Credentials
}

func (t *OAuthToken) AccessToken() string         { return t.creds.AccessToken }
func (t *OAuthToken) RefreshToken() string        { return t.creds.RefreshToken }
func (t *OAuthToken) ExpiresAt() time.Time        { return t.creds.ExpiresAt }
func (t *OAuthToken) IsExpired() bool             { return t.creds.IsExpired() }
func (t *OAuthToken) NeedsRefresh(d time.Duration) bool {
    return t.creds.NeedsRefresh(d)
}
```

## Complete Example: Protected API Server

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "time"

    "digital.vasic.auth/pkg/apikey"
    "digital.vasic.auth/pkg/jwt"
    "digital.vasic.auth/pkg/middleware"
)

// Adapt jwt.Manager to middleware.TokenValidator
type jwtValidator struct {
    manager *jwt.Manager
}

func (v *jwtValidator) ValidateToken(t string) (map[string]interface{}, error) {
    token, err := v.manager.Validate(t)
    if err != nil {
        return nil, err
    }
    return token.Claims, nil
}

// Adapt apikey store to middleware.APIKeyValidator
type apikeyValidator struct {
    store apikey.KeyStore
}

func (v *apikeyValidator) ValidateKey(key string) ([]string, error) {
    k, err := apikey.Validate(v.store, key)
    if err != nil {
        return nil, err
    }
    return k.Scopes, nil
}

func main() {
    // Setup JWT
    jwtCfg := jwt.DefaultConfig("my-secret-key-32-bytes-minimum!")
    jwtCfg.Issuer = "my-api"
    jwtManager := jwt.NewManager(jwtCfg)

    // Setup API keys
    keyGen := apikey.NewGenerator(apikey.DefaultGeneratorConfig())
    keyStore := apikey.NewInMemoryStore()
    adminKey, _ := keyGen.Generate("admin", []string{"read", "write", "admin"}, time.Time{})
    keyStore.Store(adminKey)
    fmt.Printf("Admin API Key: %s\n", adminKey.Key)

    // Validators
    tokenVal := &jwtValidator{manager: jwtManager}
    keyVal := &apikeyValidator{store: keyStore}

    // Public endpoint
    http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
        token, _ := jwtManager.Create(map[string]interface{}{
            "sub":    "user-1",
            "scopes": []string{"read", "write"},
        })
        fmt.Fprintf(w, `{"token":"%s"}`, token)
    })

    // JWT-protected endpoint
    http.Handle("/api/data", middleware.Chain(
        middleware.BearerToken(tokenVal),
        middleware.RequireScopes("read"),
    )(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := middleware.ClaimsFromContext(r.Context())
        fmt.Fprintf(w, `{"user":"%s","data":"secret"}`, claims["sub"])
    })))

    // API key-protected endpoint
    http.Handle("/api/admin", middleware.Chain(
        middleware.APIKeyHeader(keyVal, "X-API-Key"),
        middleware.RequireScopes("admin"),
    )(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, `{"status":"admin access granted"}`)
    })))

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```
