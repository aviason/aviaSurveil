# AviaCore Feed Replay, Backfill, And Recovery

This runbook governs the local `candidate-only` data-feed recovery surface. It
is `verified locally` only for the closed synthetic manifests and local
PostgreSQL fixtures named below. It is not production-ready and does not
establish AviaCore source-bound ingestion, a connected recovery drill, or a
production recovery objective.

## Scope And Owner

Owner: Data Platform / Operations

Escalation owners: AviaCore contract governance, AviaSurveil domain owner,
Security/Privacy, and Records/Legal.

The runbook covers an immutable, approval-bound replay or source-consistent
backfill run. It never writes directly to an AviaCore database, object store,
Raw Vault, or data-product relation.

## Safety Boundary

- A replay or backfill run requires a UUID approval identity, tenant and owning
  organization scope, the locked source system, and contract version `3.0.0`.
- A replay has exactly one bounded selector: event IDs or a maximum 30-day
  time window. `ACKNOWLEDGED` is never eligible for replay.
- A backfill requires an immutable source-cut identity, SHA-256 source manifest,
  original event IDs, and a cut no later than the request time. It cannot set
  occurrence time, producer revision, event payload, or a new event identity.
- Original delivery state and attempt history remain immutable. A replay/backfill
  uses its own fenced delivery lane and append-only attempts.
- Tombstoned history is never selected. A restored downstream state never
  implies producer acknowledgement; only an exact bound receipt can do that.
- Use only the direct TLS 1.3 mTLS event API; export, snapshot, direct database,
  and object-store modes are unsupported.

## Preconditions

- The locked v3 protocol/authorization identities are current in the
  AviaSurveil mirror.
- An authorized value-free request file or persisted run identity exists.
- For reconciliation, producer and AviaCore manifests are separate readable
  files with only run identity, contract version, event identity, canonical
  event digest, delivery outcome, and acknowledgement receipt digest.
- No manifest contains event payloads, Evidence bytes, filenames, person data,
  secrets, or free-form diagnostics.

## Backfill

Record the approved source-consistent backfill lane without publishing it:

```bash
data-feed-backfill approved-backfill-request.json
```

The command creates an immutable `BACKFILL` run and independent pending lanes.
Run the separate replay command only with the exact persisted run identity and
the deployment’s mounted mTLS/payload-key configuration.

## Replay

Run only the persisted immutable run. The environment requires
`AVIA_DATA_FEED_REPLAY_RUN_ID` to equal `AVIA_DATA_FEED_REPLAY_ID` so the
transport header cannot target another scope.

```bash
data-feed-replay
```

The command is one-shot and bounded. A retryable result remains visible in its
separate lane for a later approved run; it does not become a background export.

## Reconciliation

Compare two independently produced manifests. The wrapper rejects an event
that is missing, extra, digest-mutated, or at a different acknowledgement
frontier; it does not accept a count-only match.

```bash
./scripts/reconcile-aviacore-feed.sh producer-manifest.json aviacore-manifest.json
```

The checked-in fixtures prove only the synthetic local format. Before a real
two-system exercise, AviaCore must supply its exact manifest from the accepted
phase/sandbox evidence; otherwise the proof is `blocked`.

## Local Contract Check

Use the closed synthetic fixtures only for local implementation verification:

```bash
AVIA_RECOVERY_PRODUCER_MANIFEST=apps/api/tests/fixtures/datafeed/task6/producer-manifest.json \
AVIA_RECOVERY_AVIACORE_MANIFEST=apps/api/tests/fixtures/datafeed/task6/aviacore-manifest.json \
./scripts/test-aviacore-feed-recovery.sh acceptance
```

Expected output is `verified locally`, `candidate-only`, and
`production-ready: not established`. The script creates and removes its own
evidence root. It is not a claim that AviaCore Phase 2.3+, a connected
two-system recovery, real source backfill, or a production RPO/RTO has run.

## Escalation

Stop and escalate a missing/extra event, digest mismatch, receipt mismatch,
certificate issue, source-cut mismatch, tombstone attempt, unexpected queue
state, or missing AviaCore manifest. Records/Legal owns any legal-hold or
retention decision; do not alter historical events to resolve it.
