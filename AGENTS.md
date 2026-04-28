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



<!-- CONST-035 anti-bluff addendum (cascaded) -->

## CONST-035 — Anti-Bluff Tests & Challenges (mandatory; inherits from root)

Tests and Challenges in this submodule MUST verify the product, not
the LLM's mental model of the product. A test that passes when the
feature is broken is worse than a missing test — it gives false
confidence and lets defects ship to users. Functional probes at the
protocol layer are mandatory:

- TCP-open is the FLOOR, not the ceiling. Postgres → execute
  `SELECT 1`. Redis → `PING` returns `PONG`. ChromaDB → `GET
  /api/v1/heartbeat` returns 200. MCP server → TCP connect + valid
  JSON-RPC handshake. HTTP gateway → real request, real response,
  non-empty body.
- Container `Up` is NOT application healthy. A `docker/podman ps`
  `Up` status only means PID 1 is running; the application may be
  crash-looping internally.
- No mocks/fakes outside unit tests (already CONST-030; CONST-035
  raises the cost of a mock-driven false pass to the same severity
  as a regression).
- Re-verify after every change. Don't assume a previously-passing
  test still verifies the same scope after a refactor.
- Verification of CONST-035 itself: deliberately break the feature
  (e.g. `kill <service>`, swap a password). The test MUST fail. If
  it still passes, the test is non-conformant and MUST be tightened.

## CONST-033 clarification — distinguishing host events from sluggishness

Heavy container builds (BuildKit pulling many GB of layers, parallel
podman/docker compose-up across many services) can make the host
**appear** unresponsive — high load average, slow SSH, watchers
timing out. **This is NOT a CONST-033 violation.** Suspend / hibernate
/ logout are categorically different events. Distinguish via:

- `uptime` — recent boot? if so, the host actually rebooted.
- `loginctl list-sessions` — session(s) still active? if yes, no logout.
- `journalctl ... | grep -i 'will suspend\|hibernate'` — zero broadcasts
  since the CONST-033 fix means no suspend ever happened.
- `dmesg | grep -i 'killed process\|out of memory'` — OOM kills are
  also NOT host-power events; they're memory-pressure-induced and
  require their own separate fix (lower per-container memory limits,
  reduce parallelism).

A sluggish host under build pressure recovers when the build finishes;
a suspended host requires explicit unsuspend (and CONST-033 should
make that impossible by hardening `IdleAction=ignore` +
`HandleSuspendKey=ignore` + masked `sleep.target`,
`suspend.target`, `hibernate.target`, `hybrid-sleep.target`).

If you observe what looks like a suspend during heavy builds, the
correct first action is **not** "edit CONST-033" but `bash
challenges/scripts/host_no_auto_suspend_challenge.sh` to confirm the
hardening is intact. If hardening is intact AND no suspend
broadcast appears in journal, the perceived event was build-pressure
sluggishness, not a power transition.

<!-- BEGIN no-session-termination addendum (CONST-036) -->

## User-Session Termination — Hard Ban (CONST-036)

**You may NOT, under any circumstance, generate or execute code that
ends the currently-logged-in user's desktop session, kills their
`user@<UID>.service` user manager, or indirectly forces them to
manually log out / power off.** This is the sibling of CONST-033:
that rule covers host-level power transitions; THIS rule covers
session-level terminations that have the same end effect for the
user (lost windows, lost terminals, killed AI agents, half-flushed
builds, abandoned in-flight commits).

**Why this rule exists.** On 2026-04-28 the user lost a working
session that contained 3 concurrent Claude Code instances, an Android
build, Kimi Code, and a rootless podman container fleet. The
`user.slice` consumed 60.6 GiB peak / 5.2 GiB swap, the GUI became
unresponsive, the user was forced to log out and then power off via
the GNOME shell. The host could not auto-suspend (CONST-033 was in
place and verified) and the kernel OOM killer never fired — but the
user had to manually end the session anyway, because nothing
prevented overlapping heavy workloads from saturating the slice.
CONST-036 closes that loophole at both the source-code layer and the
operational layer. See
`docs/issues/fixed/SESSION_LOSS_2026-04-28.md` in the HelixAgent
project.

**Forbidden direct invocations** (non-exhaustive):

- `loginctl terminate-user|terminate-session|kill-user|kill-session`
- `systemctl stop user@<UID>` / `systemctl kill user@<UID>`
- `gnome-session-quit`
- `pkill -KILL -u $USER` / `killall -u $USER`
- `dbus-send` / `busctl` calls to `org.gnome.SessionManager.Logout|Shutdown|Reboot`
- `echo X > /sys/power/state`
- `/usr/bin/poweroff`, `/usr/bin/reboot`, `/usr/bin/halt`

**Indirect-pressure clauses:**

1. Do not spawn parallel heavy workloads casually; check `free -h`
   first; keep `user.slice` under 70% of physical RAM.
2. Long-lived background subagents go in `system.slice`. Rootless
   podman containers die with the user manager.
3. Document AI-agent concurrency caps in CLAUDE.md.
4. Never script "log out and back in" recovery flows.

**Defence:** every project ships
`scripts/host-power-management/check-no-session-termination-calls.sh`
(static scanner) and
`challenges/scripts/no_session_termination_calls_challenge.sh`
(challenge wrapper). Both MUST be wired into the project's CI /
`run_all_challenges.sh`.

<!-- END no-session-termination addendum (CONST-036) -->
