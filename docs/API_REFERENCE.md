# Auth Module - API Reference

Complete reference for all exported types, functions, and methods in `digital.vasic.auth`.

---

## Package `token`

```go
import "digital.vasic.auth/pkg/token"
```

### Interfaces

#### Token

```go
type Token interface {
    AccessToken() string
    RefreshToken() string
    ExpiresAt() time.Time
    IsExpired() bool
    NeedsRefresh(threshold time.Duration) bool
}
```

Generic authentication token with access/refresh capabilities and expiration checking.

- **AccessToken()** -- Returns the access token string.
- **RefreshToken()** -- Returns the refresh token string.
- **ExpiresAt()** -- Returns the token expiration time.
- **IsExpired()** -- Returns true if the token has expired.
- **NeedsRefresh(threshold)** -- Returns true if the token will expire within `threshold` duration.

#### Store

```go
type Store interface {
    Get(key string) (Token, error)
    Set(key string, token Token, ttl time.Duration) error
    Delete(key string) error
    Revoke(key string) error
}
```

Token storage with get, set, delete, and revoke operations.

- **Get(key)** -- Retrieves a token by key. Returns error if not found, revoked, or expired from store.
- **Set(key, token, ttl)** -- Stores a token with optional TTL. Zero TTL means no store-level expiration.
- **Delete(key)** -- Removes a token entry by key.
- **Revoke(key)** -- Marks a token as revoked. Revoked tokens remain in store but `Get` returns an error.

### Types

#### Claims

```go
type Claims map[string]interface{}
```

Map of claim key-value pairs with typed accessor methods.

**Methods:**

- **Subject() string** -- Returns the `"sub"` claim, or empty string.
- **Issuer() string** -- Returns the `"iss"` claim, or empty string.
- **Audience() string** -- Returns the `"aud"` claim, or empty string.
- **ExpiresAt() time.Time** -- Returns the `"exp"` claim as `time.Time`. Handles `float64`, `int64`, and `time.Time` values. Returns zero time if absent or unparseable.
- **IssuedAt() time.Time** -- Returns the `"iat"` claim as `time.Time`. Same type handling as `ExpiresAt`.
- **Get(key string) interface{}** -- Returns the raw value for the given key, or nil.
- **GetString(key string) string** -- Returns the string value for the given key, or empty string if absent or not a string.

#### SimpleToken

```go
type SimpleToken struct {
    // unexported fields
}
```

Basic implementation of the `Token` interface.

**Constructor:**

```go
func NewSimpleToken(accessToken, refreshToken string, expiresAt time.Time) *SimpleToken
```

**Methods:**

- **AccessToken() string** -- Returns the access token string.
- **RefreshToken() string** -- Returns the refresh token string.
- **ExpiresAt() time.Time** -- Returns the expiration time.
- **IsExpired() bool** -- Returns true if expired. Returns false if `expiresAt` is zero.
- **NeedsRefresh(threshold time.Duration) bool** -- Returns true if within `threshold` of expiration. Returns false if `expiresAt` is zero.

#### InMemoryStore

```go
type InMemoryStore struct {
    // unexported fields
}
```

Thread-safe in-memory `Store` implementation with TTL support.

**Constructor:**

```go
func NewInMemoryStore() *InMemoryStore
```

**Methods (implements Store):**

- **Get(key string) (Token, error)** -- Returns error for missing, revoked, or store-expired entries.
- **Set(key string, token Token, ttl time.Duration) error** -- Zero TTL means no store-level expiration.
- **Delete(key string) error** -- Returns error if key not found.
- **Revoke(key string) error** -- Returns error if key not found.

**Additional methods:**

- **Cleanup() int** -- Removes all expired and revoked entries. Returns the count of removed entries.
- **Len() int** -- Returns the number of entries in the store.

---

## Package `jwt`

```go
import "digital.vasic.auth/pkg/jwt"
```

### Types

#### Config

```go
type Config struct {
    SigningMethod gojwt.SigningMethod
    Secret       []byte
    Expiration   time.Duration
    Issuer       string
}
```

JWT configuration for token creation and validation.

- **SigningMethod** -- Algorithm used to sign tokens (e.g., `gojwt.SigningMethodHS256`).
- **Secret** -- Key used to sign and verify tokens.
- **Expiration** -- Default token lifetime.
- **Issuer** -- Optional issuer claim (`"iss"`) set on created tokens.

#### Token

```go
type Token struct {
    Claims    map[string]interface{}
    ExpiresAt time.Time
    IssuedAt  time.Time
    Raw       string
}
```

Parsed JWT token with claims and metadata.

- **Claims** -- Map of all token claims.
- **ExpiresAt** -- Parsed from the `"exp"` claim.
- **IssuedAt** -- Parsed from the `"iat"` claim.
- **Raw** -- The original signed token string that was validated.

#### Manager

```go
type Manager struct {
    // unexported fields
}
```

Handles JWT creation, validation, and refresh.

**Constructor:**

```go
func NewManager(config *Config) *Manager
```

**Methods:**

- **Create(claims map[string]interface{}) (string, error)** -- Generates a signed JWT string. Automatically sets `"iat"`, `"exp"`, and `"iss"` (if configured). User claims are merged in. Accepts nil claims map.
- **Validate(tokenString string) (*Token, error)** -- Parses and validates a JWT string. Verifies signature and signing method. Returns error for invalid, expired, or tampered tokens.
- **Refresh(tokenString string) (string, error)** -- Validates the existing token, strips standard claims (`"exp"`, `"iat"`, `"iss"`), and creates a new token with fresh expiration. Returns error if the original token is invalid.

### Functions

```go
func DefaultConfig(secret string) *Config
```

Returns a `Config` with HS256 signing method and 1-hour expiration.

---

## Package `apikey`

```go
import "digital.vasic.auth/pkg/apikey"
```

### Interfaces

#### KeyStore

```go
type KeyStore interface {
    Store(key *APIKey) error
    Get(keyString string) (*APIKey, error)
    GetByID(id string) (*APIKey, error)
    Delete(keyString string) error
    List() ([]*APIKey, error)
}
```

Pluggable API key storage backend.

- **Store(key)** -- Saves an API key. Returns error if the key already exists.
- **Get(keyString)** -- Retrieves by key string. Returns error if not found.
- **GetByID(id)** -- Retrieves by UUID. Returns error if not found.
- **Delete(keyString)** -- Removes by key string. Returns error if not found.
- **List()** -- Returns all stored API keys.

### Types

#### APIKey

```go
type APIKey struct {
    ID        string
    Key       string
    Name      string
    Scopes    []string
    ExpiresAt time.Time
    CreatedAt time.Time
}
```

API key with metadata.

- **ID** -- UUID identifier.
- **Key** -- Full key string including prefix (e.g., `"sk-abcdef..."`).
- **Name** -- Human-readable name.
- **Scopes** -- Permission scopes granted to this key.
- **ExpiresAt** -- Optional expiration. Zero means no expiration.
- **CreatedAt** -- Creation timestamp.

**Methods:**

- **IsExpired() bool** -- Returns true if expired. Returns false if `ExpiresAt` is zero.
- **HasScope(scope string) bool** -- Returns true if the key has the given scope.
- **HasAllScopes(scopes []string) bool** -- Returns true if the key has all given scopes.

#### GeneratorConfig

```go
type GeneratorConfig struct {
    Prefix string
    Length int
}
```

Configuration for API key generation.

- **Prefix** -- Prepended to generated keys (e.g., `"sk-"`, `"pk-"`).
- **Length** -- Number of random bytes. The hex-encoded result is twice this length.

#### Generator

```go
type Generator struct {
    // unexported fields
}
```

Creates new API keys with configurable prefix and length.

**Constructor:**

```go
func NewGenerator(config *GeneratorConfig) *Generator
```

Accepts nil config to use defaults.

**Methods:**

- **Generate(name string, scopes []string, expiresAt time.Time) (*APIKey, error)** -- Creates a new API key using `crypto/rand`. UUID is generated via `github.com/google/uuid`. Pass zero `expiresAt` for no expiration.

#### InMemoryStore

```go
type InMemoryStore struct {
    // unexported fields
}
```

Thread-safe in-memory `KeyStore` implementation with dual-map lookup.

**Constructor:**

```go
func NewInMemoryStore() *InMemoryStore
```

**Methods (implements KeyStore):**

- **Store(key *APIKey) error** -- Returns error if key already exists.
- **Get(keyString string) (*APIKey, error)** -- O(1) lookup by key string.
- **GetByID(id string) (*APIKey, error)** -- O(1) lookup by UUID.
- **Delete(keyString string) error** -- Removes from both internal maps.
- **List() ([]*APIKey, error)** -- Returns all stored keys.

### Functions

```go
func DefaultGeneratorConfig() *GeneratorConfig
```

Returns a `GeneratorConfig` with prefix `"ak-"` and length 32.

```go
func Validate(store KeyStore, keyString string) (*APIKey, error)
```

Looks up the key in the store and checks expiration. Returns the `APIKey` if valid, or an error if not found or expired.

```go
func MaskKey(key string) string
```

Returns a masked version of the key for display. Preserves the prefix (up to the first dash) and the last 4 characters, replacing the middle with asterisks. Keys 8 characters or shorter are fully masked.

---

## Package `oauth`

```go
import "digital.vasic.auth/pkg/oauth"
```

### Interfaces

#### CredentialReader

```go
type CredentialReader interface {
    ReadCredentials(providerName string) (*Credentials, error)
}
```

Reads OAuth2 credentials for a named provider.

#### TokenRefresher

```go
type TokenRefresher interface {
    Refresh(refreshToken, endpoint string) (*Credentials, error)
}
```

Refreshes an OAuth2 token using a refresh token and endpoint URL.

### Types

#### Credentials

```go
type Credentials struct {
    AccessToken  string                 `json:"access_token"`
    RefreshToken string                 `json:"refresh_token,omitempty"`
    ExpiresAt    time.Time              `json:"expires_at"`
    Scopes       []string               `json:"scopes,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
```

OAuth2 credentials with access token, refresh token, expiration, scopes, and metadata.

**Methods:**

- **IsExpired() bool** -- Returns true if expired. Returns false if `ExpiresAt` is zero.
- **NeedsRefresh(threshold time.Duration) bool** -- Returns true if within `threshold` of expiration. Returns false if `ExpiresAt` is zero.

#### RefreshResponse

```go
type RefreshResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token,omitempty"`
    ExpiresIn    int64  `json:"expires_in"`
    TokenType    string `json:"token_type"`
    Scope        string `json:"scope,omitempty"`
}
```

Standard OAuth2 token refresh response as returned by token endpoints.

#### FileCredentialReader

```go
type FileCredentialReader struct {
    // unexported fields
}
```

Reads OAuth2 credentials from JSON files on disk.

**Constructor:**

```go
func NewFileCredentialReader(paths map[string]string) *FileCredentialReader
```

`paths` maps provider names to credential file paths.

**Methods (implements CredentialReader):**

- **ReadCredentials(providerName string) (*Credentials, error)** -- Reads and parses the JSON file for the given provider. Returns error if provider not configured, file not found, parse fails, or access token is empty.

#### HTTPTokenRefresher

```go
type HTTPTokenRefresher struct {
    // unexported fields
}
```

Refreshes tokens via HTTP POST to OAuth2 token endpoints.

**Constructor:**

```go
func NewHTTPTokenRefresher(
    client *http.Client,
    clientID string,
    extraParams map[string]string,
) *HTTPTokenRefresher
```

- `client` -- HTTP client. Nil uses a default client with 30-second timeout.
- `clientID` -- Included in requests if non-empty.
- `extraParams` -- Additional form parameters (e.g., `client_secret`).

**Methods (implements TokenRefresher):**

- **Refresh(refreshToken, endpoint string) (*Credentials, error)** -- Sends a `grant_type=refresh_token` POST request. Parses the `RefreshResponse`. If no new refresh token is returned, the original is preserved. Scopes are split from space-delimited `scope` field.

#### Config

```go
type Config struct {
    RefreshThreshold  time.Duration
    CacheDuration     time.Duration
    RateLimitInterval time.Duration
}
```

Configuration for `AutoRefresher`.

- **RefreshThreshold** -- Duration before expiration when proactive refresh triggers.
- **CacheDuration** -- How long credentials are cached before re-reading.
- **RateLimitInterval** -- Minimum time between refresh attempts per provider.

#### AutoRefresher

```go
type AutoRefresher struct {
    // unexported fields
}
```

Manages automatic credential refresh with caching and rate limiting.

**Constructor:**

```go
func NewAutoRefresher(
    reader CredentialReader,
    refresher TokenRefresher,
    config *Config,
    endpoints map[string]string,
) *AutoRefresher
```

- `config` -- Nil uses `DefaultConfig()`.
- `endpoints` -- Maps provider names to token endpoint URLs.

**Methods:**

- **GetCredentials(providerName string) (*Credentials, error)** -- Returns valid credentials, refreshing automatically if needed. Checks cache first, reads from source if stale, refreshes if expiring. Falls back to existing credentials if refresh fails but token is still valid. Returns error only if credentials are expired and refresh also fails.
- **ClearCache()** -- Removes all cached credentials.
- **ClearCacheFor(providerName string)** -- Removes cached credentials for a specific provider.

### Functions

```go
func DefaultConfig() *Config
```

Returns a `Config` with 10-minute refresh threshold, 5-minute cache duration, and 30-second rate limit interval.

```go
func NeedsRefresh(expiresAt time.Time, threshold time.Duration) bool
```

Returns true if `expiresAt` is within `threshold` of now. Returns false if `expiresAt` is zero.

```go
func IsExpired(expiresAt time.Time) bool
```

Returns true if `expiresAt` is in the past. Returns false if `expiresAt` is zero.

---

## Package `middleware`

```go
import "digital.vasic.auth/pkg/middleware"
```

### Interfaces

#### TokenValidator

```go
type TokenValidator interface {
    ValidateToken(token string) (map[string]interface{}, error)
}
```

Validates a bearer token string and returns claims.

#### APIKeyValidator

```go
type APIKeyValidator interface {
    ValidateKey(key string) ([]string, error)
}
```

Validates an API key string and returns its scopes.

### Types

#### Middleware

```go
type Middleware func(http.Handler) http.Handler
```

Function type that wraps an `http.Handler` with authentication logic.

### Constants

```go
const (
    ClaimsKey contextKey = "auth_claims"
    ScopesKey contextKey = "auth_scopes"
    APIKeyKey contextKey = "auth_api_key"
)
```

Context keys for storing authenticated values. The `contextKey` type is unexported to prevent collisions.

- **ClaimsKey** -- Stores `map[string]interface{}` set by `BearerToken`.
- **ScopesKey** -- Stores `[]string` set by `BearerToken` or `APIKeyHeader`.
- **APIKeyKey** -- Stores `string` set by `APIKeyHeader`.

### Functions

```go
func BearerToken(validator TokenValidator) Middleware
```

Creates middleware that extracts `Authorization: Bearer <token>` header, validates via the `TokenValidator`, and stores claims in context under `ClaimsKey`. If the claims contain a `"scopes"` key with a `[]string` value, those are also stored under `ScopesKey`. Returns `401 Unauthorized` JSON error on failure.

```go
func APIKeyHeader(validator APIKeyValidator, headerName string) Middleware
```

Creates middleware that extracts the API key from the specified HTTP header, validates via the `APIKeyValidator`, and stores the key under `APIKeyKey` and scopes under `ScopesKey` in context. Returns `401 Unauthorized` JSON error on failure.

```go
func RequireScopes(required ...string) Middleware
```

Creates middleware that checks the request context for all required scopes. Must be placed after `BearerToken` or `APIKeyHeader` in the chain. Returns `403 Forbidden` JSON error if scopes are missing or insufficient.

```go
func Chain(middlewares ...Middleware) Middleware
```

Combines multiple middleware into a single middleware. Applied in order: the first middleware is the outermost wrapper (executes first on requests).

```go
func ClaimsFromContext(ctx context.Context) map[string]interface{}
```

Extracts claims from request context. Returns nil if not present.

```go
func ScopesFromContext(ctx context.Context) []string
```

Extracts scopes from request context. Returns nil if not present.

```go
func APIKeyFromContext(ctx context.Context) string
```

Extracts API key from request context. Returns empty string if not present.
