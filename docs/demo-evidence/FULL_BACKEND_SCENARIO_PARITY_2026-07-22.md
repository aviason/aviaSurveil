# Full Backend Scenario Parity Evidence

Date context: 22–24 July 2026

Plan: [Full Backend Scenario Parity](../exec-plans/completed/2026-07-22-full-backend-scenario-parity-plan.md)

Historical checkpoint status: `verified locally`, `candidate-only`, `release pending`, and
`ready-for-verification`. This is not a `production-ready` claim.

## Stakeholder Closeout — 28 July 2026

The user accepted Plan 2 as a completed local `candidate-only` milestone
through the
[combined Plans 2–4 stakeholder disposition](stakeholder/PLANS2_4_STAKEHOLDER_DISPOSITION_2026-07-28.md).
Tasks 1–12 and the historical verification matrix below remain the canonical
technical basis. The historical independent Plan 2 review remains `not run`;
stakeholder acceptance does not rewrite that checkpoint.

Release remains `release pending`. Deployment and production readiness remain
`not run`. Production retention, legal hold, deletion, records operations,
identity federation, external provider, release, and operating decisions
remain deferred. No `production-ready` claim is made.

## Scope And Boundary

The Full Backend Scenario Parity plan implements the server capability boundary
required by all 86 frozen React routes. The result remains one Go modular
monolith with separate API and worker processes, one generated OpenAPI
contract, PostgreSQL module-owned stores, and build-time-separated demo and HTTP
web profiles.

The final inventory is:

- 75 OpenAPI paths, 81 operations, 79 non-health operations, 160 schemas, and
  30 guarded mutations;
- all 80 frozen Backend methods mapped to generated transport operations;
- six non-empty OpenAPI source fragments that reproduce the bundled artifact;
- schema version 11, 25 full-platform relations, 19 SQLC stores, and 87 named
  SQLC queries; the six new persistent module stores own 46 of those queries;
- 20 declared domain or bounded Go packages, with cross-module workflows
  coordinated through `internal/application`;
- 28 required Backend capability slices;
- 86 routes available in exactly two build profiles, eight roles, and 108
  unique screen/action pairs;
- 10 shared scenario families with 45 required normalized proofs.

The HTTP artifact contains no mock, seed, or test-profile input. The normal
OIDC/full API exposes no reset or seed route. The reset command remains an
out-of-process, fail-closed test-profile-only command.

## Task Results

1. Task 1 is `verified locally`: deterministic OpenAPI sources, generated Go
   and TypeScript transports, examples, closed response schemas, role metadata,
   mutation guards, and drift checks pass.
2. Task 2 is `verified locally`: forward-only migrations through schema version
   11, module-owned SQLC stores, immutable/version constraints, scoped indexes,
   and transactional idempotency/audit/change/outbox links pass live PostgreSQL
   checks.
3. Task 3 is `verified locally`: identity, profile, settings, role,
   organization, session, CSRF, and OIDC authority pass focused and full race
   checks.
4. Task 4 is `verified locally`: routine and Ad Hoc planning, Finance/GM
   release, Lead preparation, team/question assignment, inspection packages,
   materialization, and withheld notice pass direct and HTTP scenarios.
5. Task 5 is `verified locally`: question/template drafting, immutable
   publication and Audit/package snapshots, and checklist save/submit/reopen
   preserve the frozen command boundary.
6. Task 6 is `verified locally`: Potential Finding, Observation, CAP, Evidence,
   exact scan-clean gating, explicit closure authority, version immutability,
   organization isolation, and separate public/internal comments pass.
7. Task 7 is `verified locally`: Preliminary and Final Report authority,
   immutable versions, DM → GM → Executive Director chains, private document
   jobs, Auditee-safe issue/preview, and fail-closed download authorization
   pass.
8. Task 8 is `verified locally`: object-scoped communications, Internal CAA
   Note separation, immutable attachments, recipient-only notifications,
   reminders, retry state, and authorized calendar projections pass.
9. Task 9 is `verified locally`: advisory-only risk, SSP/USOAP and CAP
   effectiveness projections, administration, audit log, user lifecycle, and
   Inspector Assistant draft-without-mutation behavior pass.
10. Task 10 is `verified locally`: all 28 capability slices are required and
    all 86 routes are active in both demo and HTTP only after their aggregate
    registry, direct-load, artifact, and live contract gates pass.
11. Task 11 is `verified locally`: the same implementation records exact
    normalized MockBackend and HttpBackend transcripts for 10 scenario families
    and all 45 required proofs, including negative mutation tests.
12. Task 12 is `verified locally`: the exact required final matrix passed from
    fresh runs, the evidence surfaces are synchronized, and task-owned browser,
    test-output, and scoped-service residue was cleaned.

## Final Verification Matrix

Only the final fresh green runs below count.

| Command or gate | Literal result |
|---|---|
| `npm --prefix apps/web ci` | passed; 158 packages installed |
| `./scripts/check-contracts.sh` | passed; 15/15 contract tests, deterministic bundle and generated drift clean |
| `./scripts/check-sqlc.sh` | passed; SQLC drift clean |
| `node api/openapi/tests/contract-examples.test.mjs` | passed; 14/14 |
| configured `go -C apps/api test -race -p 1 -count=1 ./...` | passed; integration package completed in 48.993s |
| `npm --prefix apps/web run typecheck` | passed |
| `npm --prefix apps/web test` | passed; 621/621 across 58/58 files |
| `node --test tests/*.test.js tests/parity/react-legacy-parity.test.mjs` | passed; 108/108 |
| `npm --prefix apps/web run build:demo` | passed |
| `npm --prefix apps/web run build:http` | passed |
| `npm --prefix apps/web run check:app-shell` | passed for demo and HTTP; 144 files and 76 assets each |
| `node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http` | passed; 144 files and 152 build inputs |
| `node apps/web/scripts/assert-parity-boundary.mjs` | passed; 86 routes and two build profiles |
| `npm --prefix apps/web run test:e2e:mock` | passed; 35/35 |
| `./scripts/test-http-profile.sh` | passed; Go race, 15/15 contracts, SQLC, 621/621 web, live Backend 19/19, Mock 35/35, HTTP 17/17, 258/258 visible-action inventories, and worker/outbox observability |
| `./scripts/test-http-oidc-profile.sh` | passed; 1/1 with no unhandled logout rejection |
| `npm --prefix apps/web run test:e2e:offline` | passed; 7/7 |
| `git diff --check` | passed |

The full HTTP profile's accepted fresh run also completed its integration
package in 50.615s and removed its scoped PostgreSQL, Keycloak, MinIO, volumes,
and network.

## Transient And Corrected Failures

These results are not counted as passes:

- initial sandboxed local-port attempts were denied by the execution
  environment;
- an initial escalated Go run lacked the pinned local OIDC environment and
  failed before the configured final run;
- an initial sandboxed Mock Playwright run could not bind the isolated Vite
  port;
- a worker-active test-profile reset exposed PostgreSQL `40P01` at `TRUNCATE`.
  A focused RED proved the missing behavior. The minimum GREEN retries only
  `40P01`, at most three attempts with 25/50 ms context-aware backoff; focused
  and full race/profile gates then passed;
- the first OIDC pass logged an unhandled second logout under React
  `StrictMode`. A focused RED expected one logout but observed two. The minimum
  GREEN adds a component-lifetime one-shot guard and truthful caught failure
  state; the focused test, 621-test web suite, and final OIDC 1/1 run passed
  without the rejection.

The final review also found stale evidence counts (`24` relations and `45`
new-module queries) and an incorrect Plan 1 standalone-baseline label. Focused
documentation REDs reproduced both mismatches. The canonical evidence now
records the final 25-relation/46-new-query inventory and preserves the Plan 1
baseline gap without changing either hash or baseline.

## Authority And Privacy Verdict

The fresh matrix preserves Potential Finding authority; immutable
checklist/CAP/Evidence/report/document versions; Auditee organization
isolation; separate `Comment to Auditee` and `Internal CAA Note`; latest exact
scan-clean Evidence gating; CAP-is-not-closure; explicit report and closure
authority; offline recovery; and advisory-only risk and assistant behavior.

The main-agent specification-compliance review found no remaining Critical or
Important findings after the documentation corrections above. A separate
main-agent code-quality review found no remaining Critical or Important
findings after the bounded reset retry and one-shot OIDC logout corrections.
These are self-reviews and are not independent. Independent Plan 2 review is
`not run`.

## Preserved Plan 1 Gaps

The accepted Plan 1 handoff is not rewritten:

- the one-shot visual result remains literally 71/259 passed and 188/259
  failed: primitive 1/1 and route pairs 70/258;
- reports include 157 decoded-pixel, 69 semantic-substring, and 10 target-size
  failures with overlapping categories;
- target-size defects were subsequently corrected and the accessibility
  matrix passed 258/258, but the one-shot visual matrix was not rerun;
- complete comparison-by-comparison manual review is `not run`;
- standalone baseline integrity is `not verified`: the accepted manifest
  records
  `sha256:92a8ab06da1f87fd9e84b45b35fa5c3dc58aa78a6eb7f6f9c9652731e8f74967`,
  while the user-edited canonical audit hashes to
  `sha256:0ab4c60febb6d95f852f1aae2d540cb678b61c0f7111ba06f424c301325f4f9c`.

No failed comparison was converted into a pass, no accepted baseline was
replaced, and the root HTML/CSS/JavaScript oracle was not modified for parity.

## Historical Publication And Release Status

The following values describe the Plan 2 checkpoint before the 28 July 2026
stakeholder disposition:

- Independent Plan 2 review: `not run`.
- Stakeholder verification: `not run`.
- Staging: completed selectively for Plan 2; unrelated user-owned changes were
  excluded.
- Commit: explicitly authorized after verification; this evidence is included
  in the same final local commit.
- Push: `not run`.
- Deployment: `not run`.
- Plan 3 and Plan 4: `not run`.
- Artifact: `candidate-only`.
- Release: `release pending`.
- Production: not `production-ready`.
