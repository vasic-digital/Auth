# Architecture -- Auth

## Purpose

Generic, reusable Go module for authentication and authorization. Provides JWT token management, API key generation and validation, OAuth2 credential management with auto-refresh, HTTP authentication middleware, and token storage with TTL support.

## Structure

```
pkg/
  jwt/          JWT token creation, validation, and refresh with configurable signing methods
  apikey/       API key generation with configurable prefixes, validation, and pluggable storage
  oauth/        Generic OAuth2 credential management with file-based reading and HTTP token refresh
  middleware/   HTTP authentication middleware for Bearer tokens, API keys, and scope validation
  token/        Token interface, Claims map with helpers, and in-memory store with TTL support
```

## Key Components

- **`jwt.Manager`** -- Create, Validate, and Refresh JWT tokens with configurable signing (HMAC, RSA)
- **`apikey.Generator`** -- Generate API keys with configurable prefixes and lengths
- **`apikey.KeyStore`** -- Pluggable storage backend for API keys (InMemoryStore provided)
- **`oauth.AutoRefresher`** -- Combines CredentialReader and TokenRefresher for automatic OAuth2 token management with caching and rate limiting
- **`middleware.BearerToken`** / **`middleware.APIKeyHeader`** -- HTTP middleware for token and API key validation
- **`middleware.Chain`** -- Compose multiple middleware into a single handler
- **`token.Store`** -- Token storage with Get, Set, Delete, Revoke, and TTL support

## Data Flow

```
HTTP Request -> middleware.BearerToken(validator) -> Extract Authorization header
                     |                                      |
              (valid token) -> next handler          (invalid) -> 401 Unauthorized

JWT Flow: Create(claims) -> signed token string -> Validate(token) -> Claims map -> Refresh(token)

OAuth2 Flow: AutoRefresher.GetCredentials(provider) -> CredentialReader -> TokenRefresher -> cached token
```

## Dependencies

- `github.com/golang-jwt/jwt/v5` -- JWT parsing and signing
- `github.com/google/uuid` -- UUID generation for API key IDs
- `github.com/stretchr/testify` -- Test assertions

## Testing Strategy

Table-driven tests with `testify`. Tests cover JWT creation/validation/refresh, API key generation with various configurations, middleware chaining, OAuth2 auto-refresh with caching, and token store TTL expiration.
