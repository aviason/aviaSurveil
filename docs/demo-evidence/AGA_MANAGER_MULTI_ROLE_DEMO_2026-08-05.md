# Department Manager AGA Multi-Role Demo Evidence — 2026-08-05

## Scope and result

`verified locally`

```text
interactive local-preprod multi-role AGA demo; verified locally
```

The result remains `candidate-only`, `release pending`, and
`production-ready: not established`. No commit, push, deployment, upload,
publication, production mutation, or external-system action was performed.

The connected run used a disposable local-preprod namespace and the accepted
sealed AGA source package. Original question text is not retained in this
evidence file or in the workspace evidence artifact.

## Immutable evidence bindings

| Evidence | Digest or outcome |
|---|---|
| Sealed source input | `sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2` |
| Generated contract input | `sha256:d84665d22359bccf28e47657ebf2d4e99551cb91e09d9f443a2db770c6631ee6` |
| Happy-path ledger | `sha256:b6dc258a0658f13c0a0fa8288b4ed1fd868903f63fe7e9f42039c4fecc5391b3` |
| Fault-matrix ledger | `sha256:b0e74440a647ad5ca31607f5f37a921f806faae4260a42894eb2ebd85547e709` |
| Happy-path phases | 14 completed |
| Fault cases | 4 completed |
| Final residue | 0 |

## Connected happy path

- The Department Manager setup receipt proved exactly 1,310 unique candidate
  questions across 53 pages, with bounded 25-row body pages and digest-bound
  text projection.
- The browser scenario discovered 7 checks and executed 17 tests with text
  media retention off. Browser privacy-leak count was 0 and the auth callback
  boundary matched.
- Server-issued batch previews were capped at 500. The deterministic demo
  subset was selected explicitly; no candidate was implicitly approved or
  published.
- Setup, readiness, recommendation, and inspection release used server-owned
  current scope, provider/target/profile eligibility, role bindings,
  generation, Draft, taxonomy, and run pins. Included ineligible candidates
  fail closed without a write.
- Current recommendation and inspection reloads were verified. Replay,
  compare-and-swap conflict, role denial, organization denial, old-inspection
  denial, and reset replay were verified.
- The finalizer recorded 10 lifecycle commands, `findingState=CLOSED`,
  `capState=ACCEPTED`, `evidenceState=ACCEPTED`,
  `closureBasis=EVIDENCE_VERIFIED`, and `finalState=COMPLETED`.
- Manager terminal facts were generation `2` with one active and one reset
  generation, 41 Draft rows, two seals, one lifecycle stream, 10 lifecycle
  events, one reset tombstone, and zero loader/exporter login capability.
- Canonical forbidden-system delta was 0 and sealed-overlay delta after seal
  was 0. Workspace roles retained no overlay access.

## Fault matrix

The four receipt-gap cases completed `CLEANED` with zero duplicate effects and
zero residue:

| Case | Verified outcome |
|---|---|
| Inherited base receipt gap | stored receipt replayed safely; one effect; no duplicate |
| Workspace transaction receipt gap | missing receipt recreated after effect; one effect; no duplicate |
| Concurrent token reservation | exactly one winner; loser effect count 0 |
| Cleanup receipt gap | one cleanup effect; no duplicate |

The recovery ledger reported `resume=verified`, authority consumption was
complete, and final cleanup reported `residue=0`.

## Aggregate verification commands

The following gates passed with fresh local output:

```text
focused Go packages: passed
tagged preproddemo Go packages: passed
OpenAPI generation, closed-schema contract tests, and contracts-check: passed
focused frontend/typecheck: 7 files, 49 tests passed
demo and HTTP builds plus demo/HTTP artifact scans: passed
boundary tests and AGA Playwright discovery: 12 root checks passed; 17 tests discovered
artifact scanner tests: 5 passed
full Vitest: 90 files, 753 tests passed
full root Node regression: 103 tests passed
connected happy path: 14 phases, 17 browser tests, residue 0
connected fault matrix: 4 cases, residue 0
harness documentation smoke, reference scan, and git diff --check: passed
```

The workspace machine-readable summary is
`docs/demo-evidence/AGA_MANAGER_MULTI_ROLE_DEMO_2026-08-05.json`. Its summary
check is read-only and passed without mutation.

## Acceptance criteria coverage

| # | Fresh evidence |
|---:|---|
| 1 | 1,310 unique candidates, 53 pages, bounded Manager browser scenario |
| 2 | body digest projection and identity digest equality verified |
| 3 | 25-row body limit, bounded page cache, and media retention off verified |
| 4 | role/organization denials and browser privacy-leak count 0 verified |
| 5 | server-issued digest-bound previews with 500-item cap verified |
| 6 | readiness event, simulation-only Draft state, and ineligible fail-closed path verified |
| 7 | server-owned setup/current scope and release pins verified |
| 8 | immutable ordered inspection selection and no silent recommendation drop verified |
| 9 | assigned Inspector/Lead transient text path and Auditee privacy projection verified |
| 10 | Manager → Inspector → Lead → CAA Reviewer → Auditee lifecycle verified |
| 11 | CAP accepted separately; Evidence verification plus `CLOSE` produced `CLOSED` |
| 12 | persistence, telemetry, artifact, log, and retained-media boundary scans passed |
| 13 | zero overlay grants and no normal/canonical candidate capability verified |
| 14 | tagged pool ownership/close-on-error tests, zero canonical delta, and terminal facts passed |
| 15 | current-object reload, replay, CAS conflict, and uniqueness protections passed |
| 16 | 1440, 1024, and 390-width browser checks passed without console/overflow failures |
| 17 | isolated start, recovery, presentation, and stop procedure recorded in the runbook |
| 18 | final process/Compose/browser residue count 0; label remains `verified locally` |

## Limitations

Real source-owner attestation, legal interpretation, real Department Manager
approval, production identity, deployment, external delivery, real-device
coverage, and production readiness are `not run` and are not established by
this local evidence.
