# Contributing to Auth Module

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git with SSH access configured

### Clone and Setup

```bash
git clone <ssh-url>/Auth.git
cd Auth
go mod download
```

### Verify the Build

```bash
go build ./...
go test ./... -count=1 -race
```

## Development Workflow

### Branch Naming

Create a branch from `main` using the appropriate prefix:

| Prefix | Purpose |
|--------|---------|
| `feat/` | New feature or capability |
| `fix/` | Bug fix |
| `refactor/` | Code restructuring without behavior change |
| `test/` | Adding or improving tests |
| `docs/` | Documentation changes |
| `chore/` | Tooling, dependencies, CI changes |

Example: `feat/jwt-rs256-support`, `fix/oauth-cache-race`

### Commit Messages

Follow Conventional Commits format:

```
<type>(<scope>): <description>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`

Scopes: `jwt`, `apikey`, `oauth`, `middleware`, `token`

Examples:
- `feat(jwt): add RS256 signing support`
- `fix(oauth): prevent race condition in AutoRefresher cache`
- `test(apikey): add benchmark for key generation`
- `refactor(middleware): extract common header parsing`

### Code Quality Checks

Run before every commit:

```bash
# Format code
gofmt -w .
goimports -w .

# Vet for common issues
go vet ./...

# Run tests with race detection
go test ./... -count=1 -race

# Run benchmarks
go test -bench=. ./...

# Generate coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## Code Style

### General Rules

- Follow standard Go conventions and [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` formatting (enforced)
- Line length at most 100 characters
- Group imports: stdlib, third-party, internal (separated by blank lines)

### Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Unexported | camelCase | `refreshToken` |
| Exported | PascalCase | `RefreshToken` |
| Constants | UPPER_SNAKE_CASE | `MAX_RETRY_COUNT` |
| Acronyms | All-caps | `HTTP`, `URL`, `ID`, `JWT`, `API`, `TTL` |
| Receivers | 1-2 letters | `m` for Manager, `s` for Store, `g` for Generator |

### Error Handling

- Always check errors
- Wrap with context: `fmt.Errorf("failed to validate token: %w", err)`
- Use `defer` for cleanup
- Return errors, do not panic

### Interfaces

- Keep interfaces small and focused
- Define interfaces in the package that uses them, unless they are the package's primary contract
- Accept interfaces, return structs

### Testing

- Use table-driven tests with `testify`
- Name tests: `Test<Struct>_<Method>_<Scenario>`
- Test both success and error paths
- Include edge cases (nil inputs, empty strings, zero times, expired tokens)
- Use `t.Parallel()` where safe

Example:

```go
func TestManager_Validate_ExpiredToken(t *testing.T) {
    cfg := DefaultConfig("test-secret")
    cfg.Expiration = -time.Hour // already expired
    mgr := NewManager(cfg)

    token, err := mgr.Create(map[string]interface{}{"sub": "user-1"})
    require.NoError(t, err)

    _, err = mgr.Validate(token)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "expired")
}
```

## Adding New Features

### Adding a New Token Type

1. Implement the `token.Token` interface in your package or a new file in `pkg/token`
2. Add comprehensive tests
3. Document in `docs/API_REFERENCE.md`

### Adding a New Storage Backend

1. Implement the relevant interface (`token.Store` or `apikey.KeyStore`)
2. Add the implementation to the appropriate package or a new sub-package
3. Add tests using real storage if applicable (unit tests may use in-memory)
4. Document usage in `docs/USER_GUIDE.md`

### Adding New Middleware

1. Follow the `func(http.Handler) http.Handler` pattern (aliased as `middleware.Middleware`)
2. Use `context.WithValue` with package-level unexported context key types
3. Provide a corresponding `FromContext` accessor function
4. Return JSON error responses with appropriate HTTP status codes
5. Add tests covering success, missing header, invalid value, and chaining scenarios

### Modifying an Interface

Interface changes are breaking changes. Before modifying:

1. Consider if an extension interface (e.g., `StoreWithCleanup`) would suffice
2. If breaking change is required, update all implementations in the module
3. Update `docs/API_REFERENCE.md`
4. Update `AGENTS.md` interface ownership table
5. Update `CLAUDE.md` key interfaces section

## Package Independence

A critical architectural rule: packages must not import each other. Each package in `pkg/` is independently usable. If you need shared types, they belong in `pkg/token` (the foundation package) or the consumer should compose at the application level.

Correct dependency direction:
```
Consumer application
  --> middleware (for HTTP auth)
  --> jwt (for JWT operations)
  --> apikey (for API key operations)
  --> oauth (for OAuth2 operations)
  --> token (for Token/Store abstractions)
```

Never:
```
jwt --> middleware  (wrong)
apikey --> token    (wrong)
oauth --> jwt       (wrong)
```

## Pull Request Process

1. Create a feature branch from `main`
2. Make changes following all conventions above
3. Ensure all tests pass: `go test ./... -count=1 -race`
4. Ensure formatting: `gofmt -l .` (should produce no output)
5. Update documentation if public API changed
6. Submit PR with a clear description of changes and motivation
7. Address review feedback

## Dependencies

### Adding Dependencies

- Minimize external dependencies
- Prefer standard library where possible
- Any new dependency must be justified in the PR description
- Test-only dependencies go in `_test.go` files

### Current Dependencies

| Dependency | Used By | Purpose |
|-----------|---------|---------|
| `github.com/golang-jwt/jwt/v5` | `pkg/jwt` | JWT parsing and signing |
| `github.com/google/uuid` | `pkg/apikey` | UUID generation |
| `github.com/stretchr/testify` | tests | Assertions |
