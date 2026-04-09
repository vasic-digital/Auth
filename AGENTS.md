# AGENTS.md - Auth Module Multi-Agent Coordination

## Module Identity

- **Module**: `digital.vasic.auth`
- **Role**: Generic authentication and authorization library
- **Packages**: `jwt`, `apikey`, `oauth`, `middleware`, `token`
- **Go Version**: 1.24+

## Agent Responsibilities

### Auth Agent

The Auth agent owns all five packages in this module. It is responsible for:

1. **Token Abstractions** (`pkg/token`) -- Defines the core `Token` interface, `Claims` map type, `Store` interface, `InMemoryStore`, and `SimpleToken` implementation. This is the foundational package that other packages depend on.

2. **JWT Management** (`pkg/jwt`) -- Handles JWT creation, validation, and refresh using `github.com/golang-jwt/jwt/v5`. Provides `Manager` with configurable `Config` (signing method, secret, expiration, issuer).

3. **API Key Management** (`pkg/apikey`) -- Generates cryptographically random API keys with configurable prefixes and lengths. Provides `KeyStore` interface, `InMemoryStore`, validation, and key masking.

4. **OAuth2 Credential Management** (`pkg/oauth`) -- Reads OAuth2 credentials from files, refreshes tokens via HTTP endpoints, and auto-refreshes with caching and rate limiting. Core types: `CredentialReader`, `TokenRefresher`, `AutoRefresher`.

5. **HTTP Middleware** (`pkg/middleware`) -- Provides `BearerToken`, `APIKeyHeader`, `RequireScopes`, and `Chain` middleware for standard `net/http` handlers. Stores validated claims, scopes, and API keys in request context.

## Cross-Agent Coordination

### Upstream Consumers

Any service that imports `digital.vasic.auth` should coordinate with the Auth agent when:

- Adding a new authentication scheme or token type
- Changing the `Token`, `Store`, `KeyStore`, `CredentialReader`, `TokenRefresher`, `TokenValidator`, or `APIKeyValidator` interfaces
- Modifying context key constants (`ClaimsKey`, `ScopesKey`, `APIKeyKey`)
- Changing JWT claim semantics or signing algorithms

### Integration Points

| Consumer | Package Used | Purpose |
|----------|-------------|---------|
| HelixAgent | `middleware`, `jwt`, `apikey` | HTTP endpoint authentication |
| LLMsVerifier | `oauth` | Provider OAuth2 credential management |
| CLI tools | `apikey` | API key generation and validation |

## Conventions

### Interface Ownership

Each interface has a single owning package:

| Interface | Owner Package | Implementors |
|-----------|--------------|-------------|
| `token.Token` | `pkg/token` | `token.SimpleToken`, external types |
| `token.Store` | `pkg/token` | `token.InMemoryStore`, external stores |
| `apikey.KeyStore` | `pkg/apikey` | `apikey.InMemoryStore`, external stores |
| `oauth.CredentialReader` | `pkg/oauth` | `oauth.FileCredentialReader`, external readers |
| `oauth.TokenRefresher` | `pkg/oauth` | `oauth.HTTPTokenRefresher`, external refreshers |
| `middleware.TokenValidator` | `pkg/middleware` | JWT adapters, external validators |
| `middleware.APIKeyValidator` | `pkg/middleware` | API key adapters, external validators |

### Dependency Direction

```
middleware --> token (context keys, claims)
middleware --> (consumers implement TokenValidator, APIKeyValidator)
jwt       --> (standalone, no internal deps)
apikey    --> (standalone, no internal deps)
oauth     --> (standalone, no internal deps)
token     --> (foundation, no deps)
```

Packages do not import each other. Each is independently usable. The `middleware` package uses context keys of type `contextKey` (unexported) and provides accessor functions (`ClaimsFromContext`, `ScopesFromContext`, `APIKeyFromContext`).

### Testing Standards

- Table-driven tests using `testify`
- Test naming: `Test<Struct>_<Method>_<Scenario>`
- Race detection: `go test -race ./...`
- No mocks in integration or E2E tests
- All exported functions and methods must have test coverage

### Error Handling

- All errors are wrapped with `fmt.Errorf("...: %w", err)` for chain inspection
- Middleware returns JSON error responses with appropriate HTTP status codes
- Token store operations return descriptive errors (not found, revoked, expired)

## Communication Protocol

When modifying this module, agents must:

1. Run `go test ./... -count=1 -race` before proposing changes
2. Verify no breaking changes to exported interfaces
3. Update CLAUDE.md if package structure or key interfaces change
4. Follow conventional commits: `feat(jwt): ...`, `fix(oauth): ...`, etc.
5. Ensure backward compatibility -- interface changes require a major version bump


## ⚠️ MANDATORY: NO SUDO OR ROOT EXECUTION

**ALL operations MUST run at local user level ONLY.**

This is a PERMANENT and NON-NEGOTIABLE security constraint:

- **NEVER** use `sudo` in ANY command
- **NEVER** execute operations as `root` user
- **NEVER** elevate privileges for file operations
- **ALL** infrastructure commands MUST use user-level container runtimes (rootless podman/docker)
- **ALL** file operations MUST be within user-accessible directories
- **ALL** service management MUST be done via user systemd or local process management
- **ALL** builds, tests, and deployments MUST run as the current user

### Why This Matters
- **Security**: Prevents accidental system-wide damage
- **Reproducibility**: User-level operations are portable across systems
- **Safety**: Limits blast radius of any issues
- **Best Practice**: Modern container workflows are rootless by design

### When You See SUDO
If any script or command suggests using `sudo`:
1. STOP immediately
2. Find a user-level alternative
3. Use rootless container runtimes
4. Modify commands to work within user permissions

**VIOLATION OF THIS CONSTRAINT IS STRICTLY PROHIBITED.**


### ⚠️⚠️⚠️ ABSOLUTELY MANDATORY: ZERO UNFINISHED WORK POLICY

NO unfinished work, TODOs, or known issues may remain in the codebase. EVER.

PROHIBITED: TODO/FIXME comments, empty implementations, silent errors, fake data, unwrap() calls that panic, empty catch blocks.

REQUIRED: Fix ALL issues immediately, complete implementations before committing, proper error handling in ALL code paths, real test assertions.

Quality Principle: If it is not finished, it does not ship. If it ships, it is finished.
