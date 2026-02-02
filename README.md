# Auth

Generic, reusable Go module for authentication and authorization.

**Module**: `digital.vasic.auth`

## Packages

- **pkg/jwt** - JWT token creation, validation, and refresh with configurable signing methods
- **pkg/apikey** - API key generation with configurable prefixes, validation, and pluggable storage
- **pkg/oauth** - Generic OAuth2 credential management with file-based reading, HTTP token refresh, auto-refresh with caching and rate limiting
- **pkg/middleware** - HTTP authentication middleware for Bearer tokens, API keys, scope validation, and middleware chaining
- **pkg/token** - Token interface, Claims map with helpers, and in-memory store with TTL support

## Installation

```bash
go get digital.vasic.auth
```

## Quick Start

### JWT Token Management

```go
import "digital.vasic.auth/pkg/jwt"

cfg := jwt.DefaultConfig("your-secret-key")
cfg.Issuer = "my-service"
manager := jwt.NewManager(cfg)

// Create a token
tokenStr, err := manager.Create(map[string]interface{}{
    "sub":  "user-123",
    "role": "admin",
})

// Validate a token
token, err := manager.Validate(tokenStr)
fmt.Println(token.Claims["sub"]) // "user-123"

// Refresh a token
newToken, err := manager.Refresh(tokenStr)
```

### API Key Authentication

```go
import "digital.vasic.auth/pkg/apikey"

gen := apikey.NewGenerator(&apikey.GeneratorConfig{
    Prefix: "sk-",
    Length: 32,
})

store := apikey.NewInMemoryStore()

key, err := gen.Generate("my-key", []string{"read", "write"}, time.Time{})
store.Store(key)

validated, err := apikey.Validate(store, key.Key)
```

### OAuth2 Credential Management

```go
import "digital.vasic.auth/pkg/oauth"

reader := oauth.NewFileCredentialReader(map[string]string{
    "github": "/path/to/github-creds.json",
})

refresher := oauth.NewHTTPTokenRefresher(nil, "client-id", nil)

ar := oauth.NewAutoRefresher(reader, refresher, nil, map[string]string{
    "github": "https://github.com/login/oauth/access_token",
})

creds, err := ar.GetCredentials("github")
```

### HTTP Middleware

```go
import "digital.vasic.auth/pkg/middleware"

// Bearer token authentication
handler := middleware.BearerToken(myValidator)(myHandler)

// API key authentication
handler := middleware.APIKeyHeader(myKeyValidator, "X-API-Key")(myHandler)

// Chain middleware
handler := middleware.Chain(
    middleware.BearerToken(myValidator),
    middleware.RequireScopes("read", "write"),
)(myHandler)
```

## Testing

```bash
go test ./... -count=1 -race
```

## License

Proprietary - All rights reserved.
