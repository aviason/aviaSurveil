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

The canonical AGA successor uses a separately named disposable project. The
obsolete donor runbooks and aliases were physically removed after Task 9
qualification. From the repository root, the task-owned operator boundary is:

```bash
scripts/start-canonical-preprod.sh
scripts/status-canonical-preprod.sh
```

The status record must report profile `aga-preprod@1.0.0`, identity namespace
`canonical-aga-preprod-exercise-v1`, donor runtime `disabled`, and external
preprod `not run`. Its local OIDC hero lifecycle is `verified locally` but
remains `candidate-only` and `release pending`; user-owned Question Review and
New Audit visual review at 1440x900, 1024x768, and 390x844 and stakeholder
acceptance are not implied by a healthy stack and must retain their literal
evidence labels. Task 9 physical donor deletion/requalification is separately
`verified locally`. External preprod is outside this plan and remains `not run`
in its separate paused ExecPlan.

The destructive Task 8 local matrix uses two unique disposable projects and a
random user-space HTTPS port. It does not reuse or stop the named public-demo
profile:

```bash
make preprod-test-fault-restart
```

The runner verifies the selected real-PostgreSQL transaction/fault/concurrency
suite, the full 1,310-question OIDC lifecycle, a stable complete-authoritative
database fingerprint before and after a cold restart, all role panels after
restart, required dependency loss as `503/not_ready`, optional dependency loss
as `200/degraded`, worker crash recovery, donor/log denial, and exact cleanup.
Record `verified locally` only after the runner prints its fingerprint and the
exit trap leaves zero task-owned containers, volumes, networks, processes, and
runtime directory. A failure must remain a failure; do not drain legitimate
undelivered local AviaCore outbox rows merely to manufacture an empty queue.

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

### Named Cloudflare Tunnel At `demo.aviasurveil.com`

The optional `preprod-cloudflare-demo-*` profile publishes the same disposable
local candidate through a remotely managed Cloudflare Tunnel whose public
hostname is exactly `https://demo.aviasurveil.com`. The application and its
data still run on this Mac; this is a public local-origin demo transport, not
an external preprod deployment. It remains `candidate-only`, `release pending`,
and Task 10 external preprod remains `not run`.

This profile requires explicit current authorization because anyone on the
Internet can reach the application login unless a separately designed
Cloudflare Access policy is present. Use only the nine synthetic
`@synthetic.invalid` identities and the disposable exercise catalog. Never run
it with real identities, regulated-party data, or operational records.

#### One-time Cloudflare dashboard setup

1. In Cloudflare, open **Networking → Tunnels** and create a remotely managed
   tunnel, for example `aviasurveil-demo-local`.
2. Add a published application route with hostname
   `demo.aviasurveil.com` and service `http://127.0.0.1:8086`. The hostname's
   DNS record must point to that tunnel. If
   `CANONICAL_PREPROD_CLOUDFLARE_DEMO_HTTP_PORT` is overridden, update the
   dashboard service port to the same exact value.
3. On the tunnel Overview page, choose **Add a replica** and copy only the
   complete `eyJ...` connector token from the install command. Depending on the
   displayed command, it is either the final argument to `service install` or
   the value after `--token`. The dashboard may visually wrap this long value;
   copy it as one uninterrupted value through its final character. Do not use
   an account API token and do not paste the complete install command.

The connector token can run that one remotely managed tunnel. It does not need
the account-wide certificate produced by `cloudflared tunnel login`, and the
repository helper never creates, edits, lists, or deletes Cloudflare account,
DNS, Access, or tunnel resources.

Cloudflare references: [create a remotely managed tunnel](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/get-started/create-remote-tunnel/),
[publish an application hostname](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/routing-to-tunnel/),
and [`--token-file` run parameters](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/configure-tunnels/run-parameters/).

#### Store or rotate the connector token

From zsh, run the Make target directly; it invokes the repository Bash helper:

```bash
make preprod-cloudflare-demo-token
```

Paste the connector token twice at the hidden terminal prompts. The helper
accepts no secret argument or environment variable, so the token does not enter
shell history. Its native Security-framework writer avoids the 128-byte ceiling
of the interactive `security -w` prompt and writes one generic-password item
with service
`com.aviasurveil360.cloudflare-tunnel` and account
`demo.aviasurveil.com`. Re-run the same target after refreshing/rotating the
tunnel token in Cloudflare. The helper decodes and validates the connector
payload without printing it; a copied command, partial value, or truncated
value fails closed.

At runtime, the detached launcher reads the item into memory and sends it to
`cloudflared` through an inherited `/dev/fd/3` pipe using `--token-file`. The
token is never written to the repository/runtime directories, command-line
arguments, logs, or process environment. This requires `cloudflared` 2025.4.0
or newer; install or update the Homebrew package before startup when needed.

#### Start, verify, inspect users, and stop

```bash
make preprod-cloudflare-demo-up
make preprod-cloudflare-demo-status
make preprod-cloudflare-demo-users
make preprod-cloudflare-demo-down
```

Startup first validates the Keychain connector token, before any image build,
then builds the local images and opens the named connector against a small
loopback placeholder. It refuses to start the application unless public
DNS resolves and `https://demo.aviasurveil.com` reaches that exact placeholder;
this detects a missing hostname route or a dashboard origin that does not match
`http://127.0.0.1:8086`. It then starts the canonical stack with the named HTTPS
origin bound consistently into Keycloak issuer/callback URLs, API CORS, Secure
cookies, and signed-object URLs. Status verifies the exact hostname, Keychain
reference, recorded process identity, local/public readiness, OIDC issuer, and
nine synthetic identities.

Stop validates ownership, disconnects the named connector before deleting the
exact local Compose project/state, and does not delete the Keychain item or any
Cloudflare dashboard/DNS configuration. The stable hostname will be unavailable
until the profile is started again. Delete or rotate the Cloudflare credential
from Cloudflare first if compromise is suspected; do not rely on stopping the
local process alone.

## AWS Private-Pilot Production Candidate

The dedicated `deploy/aws-private-pilot/compose.yaml` surface is not a local
replacement for `deploy/local`. The Task 8 target contains exactly gateway,
API, consolidated worker, and Keycloak plus bounded database-bootstrap,
migration, and Keycloak-bootstrap jobs. React assets are embedded in the
gateway, reminders are worker-owned, and PDF rendering is native Go. Run only
its offline contracts during local preparation:

```bash
./scripts/test-aws-private-pilot-compose.sh static
./scripts/check-aws-private-pilot-infrastructure.sh
./scripts/test-aws-private-pilot-release.sh
```

Systemd owns production render/start/health/drain/stop after a separately
authorized release installs a `0600` manifest, runtime environment, reviewed
RDS CA bundle, and secret-file references. The Compose unit starts first. A
separate connector unit materializes the exact SSM SecureString into a
connector-UID `0400` file, rejects the Terraform placeholder, and then runs the
digest-bound ARM64 `cloudflared` container with IPv6 edge mode. A timer
publishes `CloudflaredTunnelHAConnections`; fewer than four or missing metrics
fails health. Authorized shutdown stops the health timer and Tunnel before
draining workers and stopping Compose so no new browser traffic arrives.

The gateway is fixed to `127.0.0.1:8080`; the EC2 security group has zero
ingress. `cloudflared` is not a Compose service and its host-network use is the
only approved exception. Do not start any of these units from this runbook.
Every future operator-side AWS command and Terragrunt provider must select
`avia` explicitly in `eu-central-1`; `default` or an omitted profile fails
closed. EC2 containers do not use a named AWS profile and retain
instance-profile access only. The recorded AWS read-only Task 7 wave does not
authorize provider planning, token population, deployment, traffic, or
external health; those remain `not run`.
