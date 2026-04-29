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


### ⚠️⚠️⚠️ ABSOLUTELY MANDATORY: ZERO UNFINISHED WORK POLICY

NO unfinished work, TODOs, or known issues may remain in the codebase. EVER.

PROHIBITED: TODO/FIXME comments, empty implementations, silent errors, fake data, unwrap() calls that panic, empty catch blocks.

REQUIRED: Fix ALL issues immediately, complete implementations before committing, proper error handling in ALL code paths, real test assertions.

Quality Principle: If it is not finished, it does not ship. If it ships, it is finished.

<!-- BEGIN host-power-management addendum (CONST-033) -->

## Host Power Management — Hard Ban (CONST-033)

**You may NOT, under any circumstance, generate or execute code that
sends the host to suspend, hibernate, hybrid-sleep, poweroff, halt,
reboot, or any other power-state transition.** This rule applies to:

- Every shell command you run via the Bash tool.
- Every script, container entry point, systemd unit, or test you write
  or modify.
- Every CLI suggestion, snippet, or example you emit.

**Forbidden invocations** (non-exhaustive — see CONST-033 in
`CONSTITUTION.md` for the full list):

- `systemctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot|kexec`
- `loginctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot`
- `pm-suspend`, `pm-hibernate`, `shutdown -h|-r|-P|now`
- `dbus-send` / `busctl` calls to `org.freedesktop.login1.Manager.Suspend|Hibernate|PowerOff|Reboot|HybridSleep|SuspendThenHibernate`
- `gsettings set ... sleep-inactive-{ac,battery}-type` to anything but `'nothing'` or `'blank'`

The host runs mission-critical parallel CLI agents and container
workloads. Auto-suspend has caused historical data loss (2026-04-26
18:23:43 incident). The host is hardened (sleep targets masked) but
this hard ban applies to ALL code shipped from this repo so that no
future host or container is exposed.

**Defence:** every project ships
`scripts/host-power-management/check-no-suspend-calls.sh` (static
scanner) and
`challenges/scripts/no_suspend_calls_challenge.sh` (challenge wrapper).
Both MUST be wired into the project's CI / `run_all_challenges.sh`.

**Full background:** `docs/HOST_POWER_MANAGEMENT.md` and `CONSTITUTION.md` (CONST-033).

<!-- END host-power-management addendum (CONST-033) -->



## Sixth Law — Real User Verification (Anti-Pseudo-Test Rule)

> Inherits from the root project's Anti-Bluff Testing Pact and the cross-project
> universal mandate (CONST-035). Submodule rules below are additive, never
> relaxing.

A test that passes while the feature it covers is broken for end users is the
most expensive kind of test in this codebase — it converts unknown breakage into
believed safety. This has happened in consuming projects before: tests and
Integration Challenge Tests executed green while large parts of the product
were unusable on a real device. That outcome is a constitutional failure, not a
coverage failure, and it MUST NOT recur in any module that depends on or is
depended on by this one.

Every test added MUST satisfy ALL of the following. Violation of any of them is
a release blocker, irrespective of coverage metrics, CI status, reviewer
sign-off, or schedule pressure.

1. **Same surfaces the user touches.** The test must traverse the production
   code path the user's action triggers, end to end, with no shortcut that
   bypasses real wiring.

2. **Provably falsifiable on real defects.** Before merging, the author MUST
   run the test once with the underlying feature deliberately broken (throw
   inside the function, return the wrong row, return the wrong status) and
   confirm the test fails with a clear assertion message. The PR description
   MUST state which deliberate break was used and what failure the test
   produced. A test that cannot be made to fail by breaking the thing it claims
   to verify is a bluff test by definition.

3. **Primary assertion on user-visible state.** The chief failure signal MUST
   be on something a real consumer could see or measure: rendered output,
   persisted database row, HTTP response body / status / header, file written
   to disk, packet on the wire. "Mock was invoked N times" is a permitted
   secondary assertion, never the primary one.

4. **Integration / Challenge tests are the load-bearing acceptance gate.** A
   green Challenge Test means a real consumer can complete the flow against
   real services — not "the wiring compiles". A feature for which a Challenge
   Test cannot be written is, by definition, not shippable.

5. **CI green is necessary, not sufficient.** Before any release tag is cut, a
   human (or a scripted black-box runner) MUST have exercised the feature
   end-to-end and observed the user-visible outcome.

6. **Inheritance.** This rule applies recursively to every consumer of this
   submodule. Consumer constitutions MAY add stricter rules but MUST NOT relax
   this one.

---

## Lava Sixth Law inheritance (consumer-side anchor, 2026-04-29)

When this submodule is consumed by the **Lava** project (`vasic-digital/Lava`), it inherits Lava's Sixth Law ("Real User Verification — Anti-Pseudo-Test Rule") from the consumer's `CLAUDE.md`. Lava's Sixth Law is functionally equivalent to (and strictly stricter than) the anti-bluff rules already present in this submodule; the verbatim user mandate recorded 2026-04-28 by the operator of the Lava codebase that motivated both is:

> "We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used! This MUST NOT be the case and execution of tests and Challenges MUST guarantee the quality, the completion and full usability by end users of the product! This MUST BE part of Constitution of our project, its CLAUDE.MD and AGENTS.MD if it is not there already, and to be applied to all Submodules's Constitution, CLAUDE.MD and AGENTS.MD as well (if not there already)!"

The 2026-04-29 lessons-learned addenda recorded in Lava's `CLAUDE.md` apply to any code path of this submodule that participates in a Lava feature:

- **6.A — Real-binary contract tests.** Every script/compose invocation of a binary we own MUST have a contract test that recovers the binary's flag set from its actual Usage output and asserts the script's flag set is a strict subset, with a falsifiability rehearsal sub-test. Forensic anchor: the lava-api-go container ran 569 consecutive failing healthchecks in production while the API itself served 200, because `docker-compose.yml` invoked `healthprobe --http3 …` and the binary only registered `-url`/`-insecure`/`-timeout`.
- **6.B — Container "Up" is not application-healthy.** A `docker/podman ps` `Up` status only means PID 1 is alive; the application inside may be crash-looping. Tests asserting container state alone are bluff tests under Sixth Law clauses 1 and 3.
- **6.C — Mirror-state mismatch checks before tagging.** "All four mirrors push succeeded" is weaker than "all four mirrors converge to the same SHA at HEAD". `scripts/tag.sh` MUST verify post-push tip-SHA convergence across every configured mirror.

Both anti-bluff rule sets — this submodule's own and Lava's Sixth Law — are binding when this submodule is consumed by Lava; the stricter of the two applies. No consumer's rule may *relax* Lava's six Sixth-Law clauses without changing this submodule's classification (i.e. demoting it from Lava-compatible).

