# AviaCore Data-Feed Coverage Register

## Status and claim boundary

This is the Task 1, local `candidate-only` source-to-contract register for the
AviaCore feed. It is `verified locally` only when
`tests/aviacore-data-feed-coverage.test.mjs` passes. It does not approve a
new event, snapshot, reference-data, or data-product contract; it does not
authorize AviaCore Phase 2.3; and it does not authorize producer code,
delivery, ingestion, training, release, or production use.

`training_allowed` is `false` and `production_ml_readiness` is `NOT_READY`.
The registered facts are not source-bound analytics evidence.

## Canonical machine-readable register

[`aviacore-data-feed-coverage.json`](aviacore-data-feed-coverage.json) is the
canonical register. It contains one explicit disposition for every current
post-`CREATE`/`ALTER` `relation.column`, and it binds that inventory to exact
migration SHA-256 fingerprints. A new, removed, renamed, or otherwise changed migration,
OpenAPI path/schema, authoritative transition/command file, or profile
manifest fails the coverage test until a new explicit register decision is
recorded.

The register is deliberately closed at the source boundary:

- A relation group names exactly one disposition and applies its field policy
  to every column in that group.
- The field policy keeps inline values closed. Unrestricted free text, Evidence
  bytes, filenames, Internal CAA Notes, investigation notes, person/contact
  values, and credentials are forbidden inline.
- Any remaining field is a Task 2 contract-extension candidate or an explicit
  operational/DQ-only fact. A data product cannot act as a transport shortcut.
- The platform tenant is authenticated independently of inspected/owning and
  actor organizations. A payload organization can never determine a tenant.

## Current disposition outcome

The frozen v1 catalog contains only the 17 predecessor lifecycle event types
listed in the JSON register. The locked v3 catalog is separately listed and is
the only event family available to the Task 4 producer. Provider catalog rows
are `approved_reference_data_contract` candidates. Task 4 corrected a prior
false mapping: read-only OpenAPI operations cannot claim that they emit an
event. The `createAuditWorkspace` mutation now reconstructs the v3
`audit.planned`/`audit.started` pair from its locked released-plan record and
the workspace it creates, in the same transaction. Its planned date remains a
closed payload fact; the event effective/known/occurred times are the actual
workspace transition time, so no future knowledge timestamp is invented.
The transition fails closed—with no business, Audit, sync, internal-outbox, or
feed row—when the local writer is absent. It also validates the exact source
inspection type before that configuration check, so an unsupported or
ambiguous value cannot bypass denial when the writer is unavailable.

The following source families are not silently treated as covered by those 17
events: identity/membership, organization/service-provider scope, planning and
approvals, assignments, template/question/reference versions, Potential
Findings and report decisions, documents/communications/notifications,
calendar/reminders, advisory risk, administration, sessions/tokens/jobs,
object pointers, sync cursors, and internal audit/outbox records. Their current
dispositions are explicit extension candidates, sensitive-restricted facts, or
operational/DQ-only facts in the register.

Task 2 must give each extension candidate its named purpose, owner, contract
family, Data Vault/data-product grain, retention/deletion rule, and
compatibility/rollback disposition before any successor contract digest is
created. Existing AviaCore v1 and v2 predecessor bytes remain immutable.

## Task 4 producer outbox boundary

Migration 22 adds the local `datafeed_*` relations as a separate immutable v3
producer-outbox boundary. They are not a second source of business truth and
are never repaired by table scraping: a producer event must be constructed
from an authorized transaction's server-owned post-transition state or the
source command remains an explicit non-event disposition. The event header is
validated against the locally locked v3 contract; only AES-GCM ciphertext,
nonce, key reference, and payload/content digests are retained locally.

Delivery state is deliberately separate from immutable event and attempt
history. Claims are fenced by a monotonically increasing lease generation;
an event cannot become acknowledged without a current lease and receipt
digest. Legal hold prevents a replay tombstone. Retention is indefinite and
immutable: no physical deletion or crypto-erasure path exists.

Task 5 adds a separately named direct-mTLS publisher worker: it reconstructs
only scoped encrypted rows, rechecks payload/canonical/content digests, uses
the locked TLS 1.3 endpoint/media contract, and persists fenced receipt or
retry/quarantine outcomes. This remains `candidate-only`; it does not
authorize connected AviaCore Phase 2.3, source-bound delivery, analytics, ML,
release, or production use.

Task 6 adds a distinct, approval-bound replay/backfill run identity and a
separate fenced delivery lane. The run snapshots the exact source event
identity, canonical digest, and original occurrence/effective/known times;
receipts can update only the replay lane and append-only replay attempts, never
the original source delivery state. The closed local reconciliation command
compares every manifest row by identity, canonical digest, outcome, and receipt
frontier and rejects missing, extra, or altered rows. Its checked-in fixtures
are synthetic local format evidence only. AviaCore admission/raw manifests,
coordinated recovery, RPO/RTO, and Phase 2.3 are `not run` and `blocked` until
the separately authorized AviaCore slice produces exact evidence.

## Completeness proofs

The Task 1 gate maintains three separate proofs:

1. The static persisted-source inventory is the exact migration relation and
   column surface bound by the migration fingerprint list.
2. The command/transition coverage proof locks the authoritative application,
   governed-checklist, and profile-manifest sources plus every OpenAPI
   operation ID. Each OpenAPI operation, literal Audit action, and literal
   internal outbox topic has exactly one named target relation, disposition,
   and future contract target; an unlinked authority fact fails the gate.
3. The profile-manifest proof reserves reconciliation for a later producer
   event/AviaCore acknowledgement run. Synthetic rows do not claim an
   unexecuted mutation branch.

Later tenant/RLS/object-policy/join/restore/reconciliation tests must include
two platform tenants with colliding local business IDs and one CAA tenant with
multiple inspected Auditee organizations. That requirement is recorded now;
it is not evidence that a feed exists.

## Verification

```bash
node --test tests/aviacore-data-feed-coverage.test.mjs
node tests/harness-docs-smoke.test.js
git diff --check
```

The Task 1 test rejects unknown, stale, omitted, or duplicate relation
dispositions, source fingerprint drift, a changed v1 event catalog, an
unregistered OpenAPI operation, forbidden inline-policy drift, and a product
that claims to substitute for transport.
