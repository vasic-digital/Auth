# CLAUDE.md - Auth Module

## Overview

`digital.vasic.auth` is a generic, reusable Go module for authentication and authorization, providing JWT management, API key authentication, OAuth2 credential management, HTTP middleware, and token utilities.

**Module**: `digital.vasic.auth` (Go 1.24+)

## Build & Test

```bash
go build ./...
go test ./... -count=1 -race
go test ./... -short              # Unit tests only
go test -bench=. ./...            # Benchmarks
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## Code Style

- Standard Go conventions, `gofmt` formatting
- Imports grouped: stdlib, third-party, internal (blank line separated)
- Line length <= 100 chars
- Naming: `camelCase` private, `PascalCase` exported, acronyms all-caps
- Errors: always check, wrap with `fmt.Errorf("...: %w", err)`
- Tests: table-driven, `testify`, naming `Test<Struct>_<Method>_<Scenario>`

## Package Structure

| Package | Purpose |
|---------|---------|
| `pkg/jwt` | JWT token creation, validation, and refresh |
| `pkg/apikey` | API key generation, validation, and storage |
| `pkg/oauth` | Generic OAuth2 credential reading, token refresh, auto-refresh |
| `pkg/middleware` | HTTP auth middleware (Bearer, API key, scopes, chaining) |
| `pkg/token` | Token interface, Claims map, in-memory store with TTL |

## Key Interfaces

- `token.Token` - Generic token with access/refresh/expiry
- `token.Store` - Token storage (Get, Set, Delete, Revoke)
- `oauth.CredentialReader` - Read OAuth2 credentials by provider name
- `oauth.TokenRefresher` - Refresh OAuth2 tokens via endpoint
- `apikey.KeyStore` - Pluggable API key storage backend
- `middleware.TokenValidator` - Validate bearer tokens
- `middleware.APIKeyValidator` - Validate API keys

## Dependencies

- `github.com/golang-jwt/jwt/v5` - JWT parsing and signing
- `github.com/google/uuid` - UUID generation for API key IDs
- `github.com/stretchr/testify` - Test assertions

## Commit Style

Conventional Commits: `feat(jwt): add RS256 signing support`


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

