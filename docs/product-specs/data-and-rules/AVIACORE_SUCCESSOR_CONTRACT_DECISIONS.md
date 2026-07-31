# AviaCore Successor Contract Decisions

Burak Karahan / owner approved this Task 2 scope on 2026-07-29 for all named
producer/domain, contract-governance, data-platform/data-product,
privacy/retention, and Data/ML roles. The canonical decision record is
[`aviacore-successor-contract-decisions.json`](aviacore-successor-contract-decisions.json).

The successor is the not-yet-created `contracts/aviasurveil-production/v3/`
root. v1 and current v2 behavioral/authorization files remain immutable
predecessor inputs. Because no contract is deployed, there is no v1/v3 overlap;
the first deployment negotiates v3 only, and later correction is forward-fix
only.

Bootstrap uses a historical event-API backfill from a source-consistent cut,
never a table dump disguised as realtime. Snapshot delivery remains disabled.
v3 must add explicit correction/supersession, complete hash projections, a
tombstone/replay-suppression event, and exhaustive branch vectors.

Canonical records are retained indefinitely and immutable. Legal hold restricts
access and processing. Thirty days after a hold is released, a tombstone may
prevent replay or publication; it does not physically delete the canonical
record. Burak Karahan / owner owns tombstone and replay-suppression decisions.

This approves scope only. It does not create v3 bytes, a behavioral digest,
authorization envelope, producer mirror, code generation, outbox, AviaCore
Phase 2.3, deployment, source-bound analytics, or ML training.

## Task 3A owner-approved closed event families

On 2026-07-29, Burak Karahan / owner approved the closed event-family detail
in the decision JSON's `common_envelope`, `question_snapshot_contract`, and
`extension_event_contracts` sections. These sections are the source of truth
for the v3 cut: all event payloads are closed, all family fields are explicitly
required or conditionally required, and every listed forbidden field remains
absent. The ordered question snapshot is an immutable, canonical array; its
only bounded prose field is `prompt` (at most 4,096 bytes). It includes the
question identity, order, mapping and citation references, verification method,
expected Evidence type codes, permitted answers, and mandatory/safety flags.

This is current authority for AviaCore Task 3A only. It authorizes neither the
AviaSurveil Task 3B mirror/code-generation slice nor AviaCore Phase 2.3.
