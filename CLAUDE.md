# CLAUDE.md - Auth Module


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# JWT create/validate/refresh, API-key generate/validate, OAuth auto-refresh, HTTP middleware
cd Auth && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v ./tests/integration/...
```
Expect: PASS; exercises `jwt.NewManager`, `apikey.NewGenerator`, `oauth.NewAutoRefresher`, and `middleware.BearerToken/APIKeyHeader` per `Auth/README.md` Quick Start.


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
- **NEVER** use `su` in ANY command
- **NEVER** execute operations as `root` user
- **NEVER** elevate privileges for file operations
- **ALL** infrastructure commands MUST use user-level container runtimes (rootless podman/docker)
- **ALL** file operations MUST be within user-accessible directories
- **ALL** service management MUST be done via user systemd or local process management
- **ALL** builds, tests, and deployments MUST run as the current user

### Container-Based Solutions
When a build or runtime environment requires system-level dependencies, use containers instead of elevation:

- **Use the `Containers` submodule** (`https://github.com/vasic-digital/Containers`) for containerized build and runtime environments
- **Add the `Containers` submodule as a Git dependency** and configure it for local use within the project
- **Build and run inside containers** to avoid any need for privilege escalation
- **Rootless Podman/Docker** is the preferred container runtime

### Why This Matters
- **Security**: Prevents accidental system-wide damage
- **Reproducibility**: User-level operations are portable across systems
- **Safety**: Limits blast radius of any issues
- **Best Practice**: Modern container workflows are rootless by design

### When You See SUDO
If any script or command suggests using `sudo` or `su`:
1. STOP immediately
2. Find a user-level alternative
3. Use rootless container runtimes
4. Use the `Containers` submodule for containerized builds
5. Modify commands to work within user permissions

**VIOLATION OF THIS CONSTRAINT IS STRICTLY PROHIBITED.**

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | none |
| Downstream (these import this module) | HelixLLM |

*Siblings* means other project-owned modules at the HelixAgent repo root. The root HelixAgent app and external systems are not listed here — the list above is intentionally scoped to module-to-module seams, because drift *between* sibling modules is where the "tests pass, product broken" class of bug most often lives. See root `CLAUDE.md` for the rules that keep these seams contract-tested.
