# Local Incident Response

This runbook covers the local `candidate-only` alert surface. It is not
production-ready; a new incident remains `not run` until its recovery checks
complete.

## Scope And Owner

Owner: Platform/Operations

Escalation owner: Release authority

Backend owns API and worker symptoms. Platform/Operations coordinates critical
dependency incidents and preserves the evidence timeline.

## Preconditions

- Confirm the alert catalog ID, severity, owner, start time, and recovery
  condition.
- Identify the exact local project, state path, profile, and HTTPS port.
- Preserve correlation IDs while excluding credentials, message bodies,
  Internal CAA Notes, and Evidence bytes.

## Symptoms

- Warning: API latency, ready outbox age, or incremental backup age exceeds its
  bounded objective.
- Critical: a required dependency is down, worker attempts are exhausted,
  outbox delay is critical, or full backup data is stale.

## Safety Boundary

- Treat alerts as engineering symptoms, not regulatory or Finding decisions.
- Diagnose required dependencies before derivative latency or worker alerts.
- Do not edit product records, audit rows, Evidence versions, or identity scope
  to make an alert resolve.

## Diagnosis

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-incident-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
./scripts/local-stack.sh status full
./scripts/local-stack.sh logs full
```

For the public readiness boundary:

```bash
export AVIA_LOCAL_HTTPS_PORT="18443"
curl --fail --silent --show-error --insecure "https://localhost:$AVIA_LOCAL_HTTPS_PORT/health/ready"
```

## Expected Output

The project inventory identifies the unhealthy service. Readiness returns a
successful response only when required dependencies are ready. Alertmanager
groups duplicate symptoms and emits a resolved notification after the catalog
recovery condition is met.

## Reversible Mitigation

Restart only the implicated stateless local service after evidence capture:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-incident-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
export AVIA_LOCAL_HTTPS_PORT="18443"
export AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:$AVIA_LOCAL_HTTPS_PORT"
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full restart api
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full up --detach --wait api
```

Use the service-specific runbook for identity, scan, worker, backup, or restore
symptoms. Stop if mitigation would alter persistent state.

## Recovery Verification

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-incident-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
./scripts/local-stack.sh check full
```

Verify the catalog recovery condition for its full duration, one resolved
notification, and no new critical symptom before labeling the incident
`verified locally`.

## Evidence Capture

Record alert ID, severity, owner, UTC timeline, correlation IDs, diagnosis
commands, selected mitigation, recovery condition, resolved notification, and
final status. Redact restricted values before attaching logs.

## Escalation

Escalate warning symptoms that cross a critical threshold immediately.
Platform/Operations coordinates critical incidents; Backend, Identity, or
Security owns the affected subsystem analysis.

## Authorization Required

Persistent-data mutation, manual queue rewriting, identity or role changes,
retention changes, production communication, release rollback, AWS action, and
any production incident claim require new explicit authorization.
