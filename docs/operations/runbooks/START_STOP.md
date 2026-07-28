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
