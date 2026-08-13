# Local Evidence Scan Recovery

This runbook covers quarantine and malware-scan processing in the local
`candidate-only` stack. It is not production-ready; incident evidence starts
as `not run`.

## Scope And Owner

Owner: Backend

Escalation owner: Security and Platform/Operations

Scope includes Evidence and inspection-attachment scan requests, the
fail-closed disabled scanner mode, immutable object metadata, and worker
delivery state.

## Preconditions

- Identify the exact project and state directory.
- Record the Evidence version or attachment ID without downloading object
  bytes.
- Preserve append-only Evidence versions and quarantine boundaries.

## Symptoms

- A scan request remains ready beyond its bounded age.
- Content scanning is disabled, so a requested object remains quarantined or
  is recorded as not run.
- A job reaches a terminal state, or a clean object is not promoted from
  quarantine.

## Safety Boundary

- Never mark an object clean manually, bypass scanning, overwrite an Evidence
  version, or move bytes outside the worker contract.
- Do not expose Evidence bytes, filenames beyond operational need, or Internal
  CAA Notes in incident evidence.
- A pending or failed scan cannot be accepted as reviewed Evidence.

## Diagnosis

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-scan-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full ps --all worker minio
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full exec --no-TTY postgres psql --username aviasurveil360 --dbname aviasurveil360 --tuples-only --no-align --command "SELECT topic, terminal_state, count(*) FROM outbox_messages WHERE topic IN ('evidence.scan_requested','inspection_attachment.scan_requested') AND delivered_at IS NULL GROUP BY topic, terminal_state ORDER BY topic, terminal_state;"
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full exec --no-TTY postgres psql --username aviasurveil360 --dbname aviasurveil360 --tuples-only --no-align --command "SELECT scan_state, count(*) FROM evidence_version_states GROUP BY scan_state ORDER BY scan_state;"
```

## Expected Output

Healthy processing drains eligible messages through the configured scanner
adapter. In the local and released candidate topology the adapter is
`disabled`: results are explicitly `not-run`, object promotion remains
fail-closed, and no pending state becomes reviewed Evidence.

## Reversible Mitigation

After recording counts and dependency health, restart only the worker:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-scan-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
export AVIA_LOCAL_HTTPS_PORT="18443"
export AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:$AVIA_LOCAL_HTTPS_PORT"
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full restart worker
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full up --detach --wait worker
```

Do not requeue terminal work or mutate scan state during this mitigation.

## Recovery Verification

```bash
make preprod-test-fault-restart
```

Require disabled-scanner/fail-closed evidence, deterministic state
transitions, worker restart recovery, immutable versions, and zero residue
before recording `verified locally`.

## Evidence Capture

Capture opaque record ID, organization scope, queue counts, scan state,
scanner mode/reason, timestamps, correlation ID, dependency health, and final
recovery status. Do not attach object bytes.

## Escalation

Escalate infected objects and signature integrity concerns to Security.
Escalate queue or worker faults to Backend and Platform/Operations. Escalate
records-policy questions to Records/Legal.

## Authorization Required

Manual requeue, scan-state mutation, quarantine release, Evidence deletion,
retention change, scanner policy change, production object access, and any
closure decision require new explicit authorization.
