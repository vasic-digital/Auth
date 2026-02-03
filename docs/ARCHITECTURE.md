# Auth Module - Architecture

## Overview

`digital.vasic.auth` is a modular authentication library composed of five independent packages. The architecture prioritizes composability, interface-driven design, and zero coupling between packages.

## Design Principles

### 1. Package Independence

Each package (`jwt`, `apikey`, `oauth`, `middleware`, `token`) is independently importable and has no internal cross-dependencies. A consumer can use `pkg/jwt` without pulling in `pkg/oauth` or `pkg/middleware`. This is achieved by:

- No imports between packages within the module
- Each package defining its own types and interfaces
- Composition happening at the consumer level, not within the library

### 2. Interface-First Design

Every extensible component is defined as an interface with a default implementation:

| Interface | Default Implementation | Purpose |
|-----------|----------------------|---------|
| `token.Token` | `token.SimpleToken` | Generic token abstraction |
| `token.Store` | `token.InMemoryStore` | Token storage with TTL |
| `apikey.KeyStore` | `apikey.InMemoryStore` | API key storage backend |
| `oauth.CredentialReader` | `oauth.FileCredentialReader` | OAuth2 credential source |
| `oauth.TokenRefresher` | `oauth.HTTPTokenRefresher` | OAuth2 token refresh |
| `middleware.TokenValidator` | (consumer-provided) | Bearer token validation |
| `middleware.APIKeyValidator` | (consumer-provided) | API key validation |

### 3. Minimal Dependencies

External dependencies are limited to:

- `github.com/golang-jwt/jwt/v5` -- JWT parsing and signing (used only by `pkg/jwt`)
- `github.com/google/uuid` -- UUID generation for API key IDs (used only by `pkg/apikey`)
- `github.com/stretchr/testify` -- Test assertions (test-only)

No web framework dependency. The `middleware` package uses only `net/http` from the standard library.

## Design Patterns

### Strategy Pattern

The Strategy pattern is used extensively to allow consumers to swap implementations at runtime.

**Token Validation Strategy**: The `middleware.TokenValidator` interface allows any token validation strategy to be plugged into `BearerToken` middleware. A consumer can implement JWT validation, opaque token lookup, or external service validation without changing the middleware.

```
middleware.BearerToken(validator TokenValidator) Middleware
                         ^
                         |
         +---------------+----------------+
         |               |                |
    JWT Validator   Opaque Token    External Auth
    (jwt.Manager)   (DB lookup)    (HTTP call)
```

**Key Storage Strategy**: The `apikey.KeyStore` interface allows any storage backend (in-memory, PostgreSQL, Redis, DynamoDB) to be used with the same `Validate` function.

**Credential Reading Strategy**: The `oauth.CredentialReader` interface supports file-based, environment variable, vault-based, or any custom credential source.

### Factory Pattern

Factory functions create configured instances with sensible defaults:

- `jwt.DefaultConfig(secret)` -- Creates a Config with HS256 and 1-hour expiration
- `apikey.DefaultGeneratorConfig()` -- Creates a GeneratorConfig with "ak-" prefix and 32-byte length
- `oauth.DefaultConfig()` -- Creates a Config with 10-minute refresh threshold, 5-minute cache, 30-second rate limit
- `apikey.NewGenerator(nil)` -- Accepts nil config to use defaults
- `oauth.NewAutoRefresher(..., nil, ...)` -- Accepts nil config to use defaults

This pattern allows quick setup while supporting full customization:

```go
// Quick setup with defaults
manager := jwt.NewManager(jwt.DefaultConfig("secret"))

// Full customization
manager := jwt.NewManager(&jwt.Config{
    SigningMethod: gojwt.SigningMethodHS384,
    Secret:        []byte("custom-secret"),
    Expiration:    24 * time.Hour,
    Issuer:        "my-service",
})
```

### Decorator Pattern

The `middleware` package implements the Decorator pattern through HTTP middleware chaining. Each middleware wraps an `http.Handler`, adding behavior without modifying the original handler.

```
Request --> BearerToken --> RequireScopes --> Handler
              (auth)         (authz)        (logic)
```

The `Chain` function composes decorators:

```go
middleware.Chain(
    middleware.BearerToken(validator),     // Layer 1: authenticate
    middleware.RequireScopes("read"),      // Layer 2: authorize
)(handler)                                // Core handler
```

Middleware are applied outside-in: the first middleware in the chain is the outermost wrapper, meaning it executes first on the request path.

### Template Method (via Interface Contracts)

The `AutoRefresher` follows a template method approach internally. Its `GetCredentials` method defines the algorithm skeleton:

1. Check cache
2. Read credentials from source (via `CredentialReader`)
3. Check if refresh is needed
4. Refresh if needed (via `TokenRefresher`)
5. Update cache

The credential reading (step 2) and token refreshing (step 4) are delegated to injected interfaces, allowing the algorithm steps to vary independently.

## Package Architecture

### pkg/token

The foundation package defining core abstractions:

- **Token interface** -- Five-method contract for any authentication token: `AccessToken()`, `RefreshToken()`, `ExpiresAt()`, `IsExpired()`, `NeedsRefresh(threshold)`
- **Claims type** -- `map[string]interface{}` with typed accessor methods for standard JWT claims (`sub`, `iss`, `aud`, `exp`, `iat`) plus generic `Get` and `GetString`
- **Store interface** -- Four-method contract for token persistence: `Get`, `Set`, `Delete`, `Revoke`
- **InMemoryStore** -- Thread-safe (`sync.RWMutex`) implementation with TTL support and `Cleanup` method for removing expired/revoked entries
- **SimpleToken** -- Basic `Token` implementation with immutable fields

### pkg/jwt

JWT-specific token management:

- **Config** -- Holds signing method (`gojwt.SigningMethod`), secret, expiration duration, and optional issuer
- **Token** -- Parsed JWT representation with Claims map, ExpiresAt, IssuedAt, and Raw string
- **Manager** -- Stateful manager that creates, validates, and refreshes JWTs. Uses `github.com/golang-jwt/jwt/v5` for all cryptographic operations

The Manager's `Validate` method enforces signing method consistency -- it rejects tokens signed with a different algorithm than configured, preventing algorithm substitution attacks.

### pkg/apikey

API key lifecycle management:

- **APIKey struct** -- Contains ID (UUID), Key (prefixed hex string), Name, Scopes, ExpiresAt, CreatedAt
- **Generator** -- Uses `crypto/rand` for cryptographically secure random key generation. Keys are formatted as `<prefix><hex-encoded-random-bytes>`
- **KeyStore interface** -- Five-method storage contract with dual lookup (by key string and by UUID)
- **InMemoryStore** -- Dual-map implementation (`byKey` and `byID`) for O(1) lookup by either key string or UUID
- **Validate function** -- Package-level function that combines store lookup with expiration checking
- **MaskKey function** -- Utility for safe display of API keys, preserving prefix and last 4 characters

### pkg/oauth

OAuth2 credential management with three layers:

1. **Reading** -- `CredentialReader` interface with `FileCredentialReader` implementation that reads JSON credential files from disk
2. **Refreshing** -- `TokenRefresher` interface with `HTTPTokenRefresher` that sends standard OAuth2 refresh_token grant requests
3. **Auto-refresh** -- `AutoRefresher` that combines reading and refreshing with caching and rate limiting

The `AutoRefresher` maintains:
- A credential cache (`map[string]*cachedCredentials`) with configurable TTL
- A rate limiter (`map[string]time.Time`) tracking last refresh attempt per provider
- Mutex-protected state for thread safety

Rate limiting prevents thundering herd problems when multiple goroutines simultaneously detect an expiring token.

### pkg/middleware

HTTP middleware for `net/http`:

- **BearerToken** -- Extracts `Authorization: Bearer <token>` header, validates via `TokenValidator`, stores claims in context
- **APIKeyHeader** -- Extracts API key from a configurable header, validates via `APIKeyValidator`, stores scopes and key in context
- **RequireScopes** -- Checks context for required scopes (set by either BearerToken or APIKeyHeader)
- **Chain** -- Composes multiple middleware functions into a single middleware
- **Context accessors** -- `ClaimsFromContext`, `ScopesFromContext`, `APIKeyFromContext` for safe extraction

Context keys use an unexported `contextKey` type to prevent collisions with other packages' context values.

## Thread Safety

All stateful types are thread-safe:

| Type | Mechanism |
|------|-----------|
| `token.InMemoryStore` | `sync.RWMutex` |
| `apikey.InMemoryStore` | `sync.RWMutex` |
| `oauth.AutoRefresher` | `sync.Mutex` |

Read-heavy workloads benefit from `RWMutex` in the store implementations, where reads (`Get`, `List`, `Len`) acquire read locks and writes (`Set`, `Store`, `Delete`, `Revoke`) acquire write locks.

The `AutoRefresher` uses a regular `Mutex` because `GetCredentials` may perform both reads and writes (cache check followed by cache update).

## Error Handling

Errors follow Go conventions:

- All errors are wrapped with `fmt.Errorf("context: %w", err)` for chain inspection with `errors.Is` and `errors.As`
- Middleware returns JSON-formatted error responses with appropriate HTTP status codes (401 Unauthorized, 403 Forbidden)
- Store operations return descriptive errors distinguishing between "not found", "revoked", and "expired" states
- The `oauth` package distinguishes between "expired and refresh failed" (fatal) and "expiring but still valid" (degraded) states

## Security Considerations

1. **Algorithm enforcement** -- `jwt.Manager.Validate` verifies the token's signing algorithm matches the configured algorithm, preventing algorithm substitution attacks (e.g., `none` algorithm)
2. **Cryptographic randomness** -- `apikey.Generator` uses `crypto/rand` for key generation, not `math/rand`
3. **Key masking** -- `apikey.MaskKey` prevents accidental logging of full API keys
4. **Context isolation** -- Middleware uses unexported context key types to prevent external code from forging authentication context values
5. **Rate limiting** -- `oauth.AutoRefresher` rate-limits refresh attempts to prevent credential endpoint abuse
6. **No secret logging** -- No package logs or returns secret material in error messages
