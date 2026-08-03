# Preprod AGA Candidate Demo Evidence — 2026-08-01

## Scope

This record covers one connected, whole-namespace disposable local-preprod
qualification of the immutable AGA candidate-demo PostgreSQL overlay. It is
`candidate-only`, `release pending`, and `production-ready: not established`.

No real governed AGA, identity, assignment, source-attestation, decision,
publication, delivery, Finding, Audit, release, or production record was
created.

## Verified locally

The `aga-demo-run-20260802-local-smoke-47` run used the exact accepted
raw-byte-free ZIP through the separate provider-free loader. Its private base
receipt, target fingerprint, configuration, load authorization, and cleanup
authorization were supplied outside repository evidence and were `0600`.

- Intent digest: `sha256:b3139c83b5fecc7e60c66b9e2c48c9e2bc8e6aa0e34315d6cf0e0bfa7200e78c`
- Authorization receipt hash: `sha256:84642e2daef5e80afd93120305a3332997aad739fe346fff7c8d1b6c3fdc1cd2`
- Result digest: `sha256:576c2941c816804a1267f337717d0fc10ffe59022db7d39e87b9861b09334815`
- Final database seal digest: `sha256:66b26d18fb784e2f896948f10bf478d81fa91211f6cdbde8395edf03a7587e95`
- Reconciliation digest: `sha256:c73a932311842918e8e39fbda3479fbf8a9b156a670dc87bc0b2852ccba72a5d`
- Cleanup tombstone digest: `sha256:4c00af1aaddfa3873e2b820104a5d114d3dfc6e80690f13a0ec98b32c4d7a773`

The connected harness proved the final in-transaction seal could be
reconstructed by a separate read-only verification. It then consumed the
separate cleanup authorization and proved a second load was rejected as
`non-replayable`.

The matrix ran with three distinct database passwords. A rollback-only positive
probe proved the normal API credential retained only the existing OIDC session
and Department Manager authentication surface and could not read the overlay.
The tagged reader read exactly one sealed package but could not access base
tables, write, or change role. The one-shot writer could not add after the
seal, delete, run DDL, or escalate.

The separate sealed-reader reconciliation proved 52 exact forms, 1,310 exact
questions, the 21/31 extraction split, 1,261/49 source split, 2,329 question
links, 174 unique sources, 14 expert blockers, fixed candidate states, null
authority fields, and the expected provisional risk distributions. The final
in-transaction reconciliation seal remained the sole database readability
receipt.

The tagged API used the ordinary HTTP React artifact and the existing OIDC
boundary after a separate disposable predecessor fixture made the nine exact
existing synthetic accounts login-capable. The 10-test matrix covered CAA
Admin, anonymous, all seven denied role families, stale access after logout,
and absent mutation/export routes. Admin UI checks ran at `1440x900`,
`1024x768`, and `390x844`; retained trace, screenshot, and video count was
zero, and the candidate payload was absent from browser persistence and
telemetry.

Before and after snapshots were byte-identical for the non-overlay PostgreSQL
table-statistics digest, Keycloak user count, Mailpit state digest, and MinIO
object-version digest. The harness performed whole-namespace Docker cleanup;
the final project container, volume, and network residue was zero.

Machine-readable and text-only local evidence is retained under
`docs/demo-evidence/preprod-aga-candidate-demo/run-20260802-local-smoke-47/`.
It contains no question text, source URL, authentication token, secret, or
private path. All 16 evidence files have mode `0600`; the post-run privacy scan
reported zero violations.

## Aggregate verification — Task 9

The aggregate gate ran from the current uncommitted workspace against the
disposable local-preprod Docker topology. Private base, authorization, target,
and cleanup file paths were intentionally not retained in repository evidence.
The public command surface was:

```bash
node --test tests/aga-candidate-preprod-demo-plan-contract.test.mjs tests/aga-candidate-preprod-demo-boundary.test.mjs tests/preprod-data-boundary.test.mjs
bash scripts/check-contracts.sh
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" go -C apps/api test -count=1 ./internal/preproddata/agacandidatedemo -run AcceptedPackage
go -C apps/api test -count=1 ./internal/preproddata/agacandidatedemo ./internal/agacandidatedemo ./internal/httpapi ./cmd/preprod-aga-candidate-demo-loader ./cmd/api
go -C apps/api test -count=1 -tags=preproddemo ./internal/agacandidatedemo ./internal/httpapi ./cmd/api
go -C apps/api build -tags=preproddemo -o "${TMPDIR:-/tmp}/avia-preprod-aga-api-task9-final" ./cmd/api
bash scripts/test-aga-candidate-demo-loader.sh
npm --prefix apps/web run typecheck
npm --prefix apps/web test -- --run aga-candidate-demo checklist-builder-page
npm --prefix apps/web test -- --reporter=dot --no-color
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
npm --prefix apps/web run test:e2e:aga-preprod -- --list
node tests/demo-boundary-smoke.test.js
bash scripts/test-normal-artifact-boundary.sh
bash scripts/test-aga-candidate-preprod-demo-connected.sh
node tests/harness-docs-smoke.test.js
git diff --check
```

Fresh results were:

- plan/boundary/preprod Node tests: 17/17 passed;
- OpenAPI example tests: 16/16 passed, with generated contracts clean;
- exact accepted package and five focused untagged Go package groups: passed;
- three tagged Go package groups and tagged build: passed, with 71 tagged Go
  tests discovered;
- provider-free loader boundary: Go package passed and 11/11 Node tests passed;
- frontend: typecheck passed, focused Vitest passed 2 files/4 tests, full Vitest
  passed 82 files/718 tests, and demo/HTTP builds passed;
- HTTP artifact scan: 148 files and 180 build inputs passed;
- Playwright discovery: 10 tests across two files; connected Run 47 executed
  all 10 and retained zero screenshots, traces, or videos;
- demo boundary, normal-artifact boundary, documentation smoke, Markdown
  whitespace/fence, evidence privacy/mode, and final residue gates: passed.

Supplemental AGA Node coverage passed 19/19 tests. The session and identity Go
packages also passed after the test runner was granted local loopback access;
the first attempt was rejected by the workspace sandbox before application
behavior ran. The final tagged-build repeat was likewise denied only at the
macOS Go cache boundary and passed unchanged with local cache access. The
initial full Vitest aggregate correctly failed because the
new layered CSS import was absent from the exact style-ownership expectation;
that expectation was corrected, its focused test passed 6/6, and the full
82-file/718-test rerun passed. Connected Run 46 stopped before role provisioning
when a provenance-varying manifest-list ID failed the code-digest comparison;
the whole namespace cleaned to zero. Run 47 instead bound the actual
provider-free loader binary digest obtained with networking disabled and is the
sole accepted connected run.

`git diff --check`, explicit plan/evidence fence and whitespace checks, the
16-file evidence privacy scan, and final task-owned Docker, process, and browser
residue checks passed. No commit, push, deployment, upload, publication, or
external-system mutation was performed.

## Follow-up demo presentation — 2026-08-02

The Admin-only candidate panel now reads every sealed question page until the
cursor is exhausted and renders all `1,310` questions in a read-only
`Synthetic Department Manager demo handoff` table. This is presentation only:
the panel creates no Department Manager identity, assignment, decision, source
attestation, publication, or governed record, and the API remains the same
five GET-only CAA Admin routes. Focused UI verification passed `2` files and
`4` tests; the full regression passed `82` files and `718` tests; typecheck,
demo/HTTP builds, and the `148`-file/`180`-input HTTP artifact scan passed.

## Not run

Production, deployment, release, real-source authority, real identity,
real-device, and external approval evidence is `not run`. This local evidence
does not establish any of those states. The result remains `candidate-only`,
`release pending`, and `production-ready: not established`.
