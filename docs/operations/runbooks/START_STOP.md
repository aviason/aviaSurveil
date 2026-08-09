# Local Stack Start And Stop

This procedure is `candidate-only`, not production-ready, and initially
`not run` for each new operator environment.

## Scope And Owner

Owner: Platform/Operations

Escalation owner: Release authority and Security

Scope is one exact task-owned local Compose project and its matching absolute
state directory.

## Preconditions

- Run from the repository root with Docker and the repository-supported Node.js
  runtime available on `PATH`.
- Choose a unique `aviasurveil360-task-*` project name and a matching state
  directory that is not shared with another stack.
- Record the selected profile and HTTPS port before startup.

## Symptoms

- A planned local stack is absent, unhealthy, or no longer required.
- A command refuses an ambiguous project or reports a state ownership mismatch.
- A service remains unhealthy after the bounded startup wait.

## Safety Boundary

- Never target a project or state directory whose ownership is unknown.
- Do not use broad Docker cleanup commands.
- The `down` command removes the exact task-owned stack and its local state,
  including local data and credentials.

## Diagnosis

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-ops-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
./scripts/local-stack.sh status full
./scripts/local-stack.sh check full
```

## Expected Output

`status` lists only the selected project. `check` reports a healthy runtime, or
the command fails with the exact unhealthy dependency. An ownership mismatch
must fail closed.

## Reversible Mitigation

Start a new task-owned stack without changing another stack:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-ops-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
export AVIA_LOCAL_HTTPS_PORT="18443"
export AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:$AVIA_LOCAL_HTTPS_PORT"
./scripts/local-stack.sh up full
```

If startup fails, inspect `status` and `logs`; preserve the state until evidence
is captured. Removing the failed stack is an explicit, scoped final action:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-ops-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
./scripts/local-stack.sh down full
```

## Recovery Verification

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-ops-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
./scripts/local-stack.sh check full
```

Record `verified locally` only when the exact profile is healthy and task-owned
cleanup, when requested, reports no remaining project resources.

## Evidence Capture

Capture project name, profile, state path, start/end UTC timestamps, command
status, failed healthcheck name, and the final literal evidence label. Do not
copy generated credentials into evidence.

## Escalation

Escalate ownership mismatch, repeated unhealthy dependencies, unexpected
published ports, or cleanup residue to Platform/Operations. Escalate suspected
secret exposure to Security.

## Authorization Required

Production start/stop, remote deployment, shared state removal, non-task-owned
resource deletion, and AWS actions require new explicit authorization.

## Canonical AGA Local-Preprod Qualification

The canonical AGA successor uses a separately named disposable project and
must not be started through the paused donor runbooks. From the repository
root, the task-owned operator boundary is:

```bash
scripts/start-canonical-preprod.sh
scripts/status-canonical-preprod.sh
```

The status record must report profile `aga-preprod@1.0.0`, identity namespace
`canonical-aga-preprod-exercise-v1`, donor runtime `disabled`, and external
preprod `not run`. Its local OIDC hero lifecycle is `verified locally` but
remains `candidate-only` and `release pending`; user-owned Question Review and
New Audit visual review at 1440x900, 1024x768, and 390x844, the remaining
negative/recovery/dependency matrix, Task 9 donor deletion/requalification,
stakeholder acceptance, and Task 10 external deployment are not implied by a
healthy stack and must retain their literal evidence labels.

### Disposable Cloudflare Quick Tunnel Profile

The optional `preprod-cloudflare-*` profile is a separate, task-owned local
qualification aid. It creates one anonymous, random `https://*.trycloudflare.com`
URL to the loopback-only HTTP gateway so public-origin OIDC, Secure cookies,
signed-object TLS, and exact CORS can be tested without changing the canonical
HTTPS profile. It is `candidate-only`, is not an external preprod deployment,
and Task 10 remains `not run`.

Because this profile makes the local disposable catalog reachable through a
public URL, obtain explicit current approval for that exposure before starting
it. Never use it with real identities, real regulated-party data, or a named
Cloudflare tunnel. It must not use Cloudflare login, tokens, DNS, Access,
routes, account configuration, AWS, or any remote preprod infrastructure.

```bash
make preprod-cloudflare-up
make preprod-cloudflare-link
make preprod-cloudflare-status
make preprod-cloudflare-users
make preprod-cloudflare-test-panels
make preprod-cloudflare-down
```

`preprod-cloudflare-link` first validates an already healthy, exact profile
and prints only its random public origin. If neither task-owned root exists,
it starts the same disposable profile and then prints that origin. It never
reuses partial or stale roots; the same explicit public-exposure approval is
required because it can start the profile.

`preprod-cloudflare-users` validates the live profile before printing the
random URL, the nine privacy-safe `@synthetic.invalid` usernames, and their
task-owned disposable password. Never copy that password into documentation or
reuse it outside this exact disposable namespace. `preprod-cloudflare-test-panels`
runs isolated headless Chromium logins for all nine accounts with Service
Workers enabled. Each context is reloaded under worker control before the test
requires visible Keycloak username/password fields, verifies the public OIDC
callback and `__Host-` Secure cookie pair, checks every role home,
and exercises Department Manager Question Review plus New Audit through the
1,310-question exact-selection review stage. It retains no screenshots, video,
trace, browser profile, or password artifact.

The start script creates new state/runtime roots only and refuses stale roots
or Compose residue. Its detached task-owned launcher sanitizes the
`cloudflared` environment, accepts exactly one bare HTTPS `trycloudflare.com`
origin, and records metadata only after local/public readiness and exact OIDC
discovery issuer checks pass. The stop script validates the project, roots,
metadata, local origin, and recorded PID identity; it removes the public
exposure first, then performs exact Compose cleanup with volumes and orphan
removal. A PID identity mismatch is an escalation condition: do not use broad
process-kill commands or manually retain the disposable state.
