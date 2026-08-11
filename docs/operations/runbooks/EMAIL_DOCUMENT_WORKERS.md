# Local Email And Document Worker Recovery

This runbook covers outbox, email delivery, and document rendering in the
local `candidate-only` stack. It is not production-ready and begins as
`not run` for each incident.

## Scope And Owner

Owner: Backend

Escalation owner: Platform/Operations and Security

Scope includes ready outbox work, bounded attempts, authenticated private
Mailpit delivery, Gotenberg rendering, and append-only document versions.

## Preconditions

- Confirm exact project ownership and identify the affected job IDs.
- Separate email, document, scan, and other outbox topics before mitigation.
- Preserve message visibility, organization scope, idempotency keys, and
  document version history.

## Symptoms

- Ready outbox age crosses warning or critical thresholds.
- Email delivery is failed or dead-lettered.
- A document render job remains pending, running, or failed.
- Worker attempts reach the critical threshold.

## Safety Boundary

- Never edit a message body, audience, organization, idempotency key, accepted
  delivery, or generated document version.
- Never turn a failed job into delivered or succeeded by direct database
  mutation.
- Local Mailpit acceptance is not external delivery evidence.

## Diagnosis

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-worker-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full ps --all worker mailpit gotenberg
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full exec --no-TTY postgres psql --username aviasurveil360 --dbname aviasurveil360 --tuples-only --no-align --command "SELECT topic, terminal_state, count(*) FROM outbox_messages WHERE delivered_at IS NULL GROUP BY topic, terminal_state ORDER BY topic, terminal_state;"
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full exec --no-TTY postgres psql --username aviasurveil360 --dbname aviasurveil360 --tuples-only --no-align --command "SELECT status, count(*) FROM notification_delivery_jobs GROUP BY status ORDER BY status; SELECT status, count(*) FROM document_render_jobs GROUP BY status ORDER BY status;"
```

## Expected Output

Healthy workers drain eligible work exactly once, preserve deduplication, store
provider/render provenance, append a new document version, and distinguish
failed retryable work from terminal work.

## Reversible Mitigation

Restart only the worker after capturing queue state:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-worker-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
export AVIA_LOCAL_HTTPS_PORT="18443"
export AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:$AVIA_LOCAL_HTTPS_PORT"
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full restart worker
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full up --detach --wait worker
```

The worker lease and idempotency contracts handle eligible work; do not alter
queue rows.

## Recovery Verification

```bash
make preprod-test-fault-restart
```

Require one accepted private email, one generated PDF with provenance, restart
recovery, append-only history, exact organization visibility, and zero residue
before recording `verified locally`.

## Evidence Capture

Capture job IDs, topic/status counts, attempt count, provider/render opaque
identifiers, correlation IDs, UTC timeline, restart result, and final state.
Exclude message bodies, recipient addresses, document bytes, and secrets.

## Escalation

Escalate queue and rendering faults to Backend. Escalate service availability
to Platform/Operations and data exposure to Security. Records/Legal owns
retention or legal-hold questions.

## Authorization Required

Manual requeue, dead-letter release, body or audience change, resend to any
external recipient, document-version mutation, production SMTP/Gotenberg use,
and retention changes require new explicit authorization.

## AWS Private-Pilot Worker Boundary

Production SMTP permits only verified implicit TLS or mandatory STARTTLS with
TLS 1.2 or newer, hostname validation, bounded timeouts, and redacted failure
codes. The selected relay must publish a reviewed AAAA result and pass the
host's forced-IPv6 certificate preflight; its security-group destination is a
reviewed IPv6 CIDR on only port 465 or 587. There is no IPv4/NAT, public
plaintext, or Mailpit fallback. The private-pilot worker uses native Go PDF
rendering with embedded fonts; no Gotenberg, Chromium, renderer URL, or
renderer container is part of the production target. Notification and document
outbox leases remain idempotent across outage, timeout, duplicate delivery,
worker restart, and the post-external-effect crash window.

The task-owned local integration gate continues to exercise authenticated
Mailpit, outage/restart recovery, exact message metadata, and the local
renderer contract. External SMTP delivery and production rendering are `not
run`. Future operator-side AWS diagnostics must use profile `avia`; runtime
containers still use no named AWS profile.
