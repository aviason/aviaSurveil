# AGA Hybrid Classification And Synthetic Demo Lifecycle — Task 9 Evidence

Date: 2026-08-04

Status: `candidate-only`; prior connected claims in this document are
invalidated pending fresh provenance-backed Task 9 gates

Product status: `candidate-only`

Release: `release pending`

Production-ready: not established

## Offline controls

- The closed forbidden-object inventory covers parsed migration tables and
  sequences, authentication-control columns, sealed predecessor-overlay
  objects, workspace objects/functions/roles/grants, and required Compose
  services/secrets.
- The separate operator-only authorization issuer uses fresh CSPRNG nonce and
  one-shot tokens, exact phase operation sets, target binding, create-only
  `0600` output, file/parent-directory `fsync`, and closed validation. The
  connected harness does not import or invoke the issuer.
- The connected harness, evidence verifier, create-once privacy-safe summary
  finalizer, and check-only summary verifier are fail-closed on missing,
  substituted, extra, or broken receipts.

## F1 — exhaustive journal/commit/publication probe

The offline writer executed all 14 journal phases crossed with all four
boundaries (`BEFORE_EFFECT`, `AFTER_EFFECT_BEFORE_TARGET_RECEIPT`,
`AFTER_TARGET_RECEIPT_BEFORE_LEDGER_PUBLICATION`, and
`AFTER_LEDGER_PUBLICATION`) as 56 real create-only private filesystem cases.
Each case used a separate child process, exercised the requested crash boundary,
recovered by either reading the stored receipt or explicitly creating the exact
missing receipt after the effect, rejected a repeated
effect with create-only semantics, validated publication, and removed its
case-owned residue. The concurrent reservation probe used two real processes and
recorded one winner with zero loser effect.

```text
caseCount=56
missingCases=0
skippedCases=0
actualEffectCaseCount=42
targetReceiptCaseCount=42
ledgerPublicationCaseCount=42
storedReceiptReplayCount=28
missingReceiptRecreationCount=14
concurrentWinnerCount=1
concurrentLoserEffectCount=0
residueCount=0
casesDigest=sha256:981ca841be5774c551d1ab9ad9862635c929f308a46797de2448bd1b43dec095
```

## F2 — connected happy path

The prior connected run is not accepted as current evidence. The predecessor
OIDC fixture reuses the nine existing synthetic identity identifiers, but its
membership revision, desired-membership synchronization, user lifecycle,
profile, and issuer control-plane changes are writes and require exact raw
receipts. The AGA workspace then
sealed 1,310 classification items, one Draft, nine authority bindings, two
provider scopes, and one append-only credential-revocation receipt. Exporter and
loader roles were `NOLOGIN`, and the sealed projection reported
`loaderRevoked=true`.

All 14 connected phases completed. The full discovered browser set was 17 tests
across five files and all 17 passed; the required targeted discovery was seven
tests across three files. The two sibling-schema barriers proved one
load-then-seal winner and one seal-then-load rejection with zero sibling
residue. Forbidden business delta and post-seal overlay delta were both zero,
overlay cleanup replay was rejected, and the disposable Compose namespace was
removed with zero containers, volumes, and networks.

## F3 — four-case recovery matrix

The previous four-case fault matrix was private-filesystem simulation and is
invalidated. A fresh F3 run must use four distinct connected task-owned target
namespaces/Compose projects with target-bound manifests, per-case authority,
target transaction receipts, interruption/resume branches, cleanup receipts,
and independent zero-residue checks.

- `INHERITED_BASE_RECEIPT_GAP`
- `WORKSPACE_TRANSACTION_RECEIPT_GAP`
- `CONCURRENT_TOKEN_RESERVATION` — one winner, zero loser effect
- `CLEANUP_RECEIPT_GAP`

Each case reached its expected terminal state with zero residue. The connected
run ended with:

```text
aga-hybrid-connected: verified locally fault-matrix cases=4 residue=0
faultLedgerDigest=sha256:aa8696531b4bf6029ad5271b7dabb80e04ca1a7a45e1b132fe864bb884c422f9
```

## Evidence handoff

The create-once privacy-safe summary was finalized only after the separate
happy-path and fault ledgers passed validation. Its aggregate values are:

```text
happyLedgerDigest=sha256:6918db01e3acd683ce07f5556bc27aa36029e57f55f1fd460753c646a2d2db15
faultLedgerDigest=sha256:aa8696531b4bf6029ad5271b7dabb80e04ca1a7a45e1b132fe864bb884c422f9
happyPhaseCount=14
faultCaseCount=4
browserTestCount=17
browserDiscoveryCount=7
browserPrivacyLeakCount=0
forbiddenBusinessDelta=0
sealedOverlayDeltaAfterSeal=0
residueCount=0
```

The create-once summary
`docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_DEMO_LIFECYCLE_2026-08-03.md`
is preserved and is not valid successor evidence for the corrected gates. No
versioned successor summary has been created yet. The current result is not a
real database, external-system, deployment, release, or production-ready
claim.

## Verification records

```text
node scripts/build-aga-hybrid-forbidden-inventory.mjs --check tests/fixtures/aga-hybrid-forbidden-object-inventory.v1.json
aga-hybrid-forbidden-inventory: ok objects=129

node --test tests/aga-hybrid-demo-workspace-boundary.test.mjs tests/aga-question-classification-candidate.test.mjs
passed

node --check scripts/issue-aga-hybrid-connected-authorization.mjs
node --check scripts/build-aga-hybrid-forbidden-inventory.mjs
node --check scripts/verify-aga-hybrid-demo-workspace-evidence.mjs
bash -n scripts/test-aga-hybrid-demo-workspace-connected.sh
passed

node scripts/check-aga-hybrid-created-files.mjs --inventory tests/fixtures/aga-hybrid-created-file-inventory.v1.json --through task8
aga-hybrid-created-files: ok through=task8 due=100 planned=107

git diff --check
passed
```

No production, external-system, real-database, deployment, commit, push, or
branch action was performed. The disposable target was cleaned by its receipt
backed final phase; private evidence ledgers were retained only for the local
Task 10 checks and then exact task-owned temporary resources were removed.
