# AviaSurveil360 Agent Harness Verification Matrix

Use the smallest level that covers the actual risk. All checks are local-only.
Do not add GitHub Actions, `.github/workflows`, hosted runners, scheduled
workflows, package-manager scripts, remote CI, or paid automation for this
private repository.

## Level 1 - Docs-Only

Use for Markdown, plan, prompt, manifest, harness, or source-index changes.

Required local checks:

```bash
git diff --check
rg -n "docs/agent-harness|agent-harness/index|output-contract|verification-matrix|entropy-cleanup" AGENTS.md MANIFEST.md docs
```

When harness docs are touched, also run:

```bash
node tests/harness-docs-smoke.test.js
```

## Level 2 - JavaScript Logic

Use for helper, data, state transition, render function, or smoke-test changes.
Run `node --check` for touched JS files and the smallest relevant smoke test.

Current syntax commands:

```bash
node --check js/data.js
node --check js/helpers.js
node --check js/approval.js
node --check js/planning.js
node --check js/checklists.js
node --check js/inspection.js
node --check js/reports.js
node --check js/views.js
node --check js/app.js
```

Current targeted smoke tests:

```bash
node tests/approval-smoke.test.js
node tests/checklist-approval-smoke.test.js
node tests/checklist-management-smoke.test.js
node tests/governance-render-smoke.test.js
node tests/inspection-execution-smoke.test.js
node tests/planning-render-smoke.test.js
node tests/planning-release-smoke.test.js
node tests/report-approval-smoke.test.js
node tests/audit-work-queue-smoke.test.js
node tests/demo-boundary-smoke.test.js
```

## Level 3 - Static Workflow

Use for user-facing click path changes.

Required checks:

- Level 2 for touched JS and targeted tests.
- Local browser click-through of the changed role path.
- Console error review for the changed path.

Use direct file opening when enough. Serve locally only when browser behavior
requires it:

```bash
python3 -m http.server 4360
```

## Level 4 - Visual Or UI

Use for layout, responsive behavior, dashboard, role home, or visual polish.

Required checks:

- Level 3 checks.
- Desktop and mobile viewport visual review.
- Confirm no incoherent overlap, hidden target content, or horizontal overflow.
- Assert against visible page content, not hidden navigation text.
- Clean up browser or GUI automation processes before reporting completion.

## Level 5 - Boundary-Sensitive

Use for auth, upload, AI, regulatory, audit-log, offline, evidence,
notification, reporting, permission, or production-readiness language.

Required checks:

- Relevant lower-level checks.
- `node tests/demo-boundary-smoke.test.js` when demo boundaries may be affected.
- Explicit review that the task did not add or claim backend, database, API,
  real auth, real upload/storage, real AI service, real regulatory ingestion,
  real notification delivery, production audit-log behavior, remote CI, or
  paid automation.

## Production-Application Candidate Lane

This lane is local-only and supplements rather than replaces the static-demo
levels above. Run only the commands supported by the explicitly authorized
slice; label unavailable later-slice gates `not run` or `blocked` rather than
silently skipping them.

For the Tasks 2-4 mock-data first executable slice:

```bash
npm --prefix apps/web ci
npm --prefix apps/web run contracts:check
npm --prefix apps/web run typecheck
npm --prefix apps/web test
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
node --test api/openapi/tests/contract-examples.test.mjs tests/parity/react-legacy-parity.test.mjs
npm --prefix apps/web run test:e2e:mock -- canonical-scenario.spec.ts
node --test tests/*.test.js
git diff --check
```

Expected: locked install succeeds; OpenAPI examples and checked generation are
clean; React type/unit/build and mock browser tests pass; the HTTP artifact has
no mock/seed or demo-public input; the legacy demo suite remains passing. Real
HTTP, Go, IndexedDB/OPFS, Service Worker/PWA, offline, sync, deployment, and
production evidence are `not run` for this slice. The result is
`candidate-only`.

For the Task 6 PWA app-shell and offline-readiness slice, add:

```bash
npm --prefix apps/web run check:app-shell
npm --prefix apps/web run test:e2e:offline
./scripts/test-http-profile.sh
```

Expected: both generated app-shell manifests match their worker version marker;
the worker has no automatic activation/cache deletion or API-response caching;
the dedicated persistent-profile browser tests pass real restart/server-stop
startup and two-client N/N-1 preservation; and the complete live HTTP profile
still passes and cleans up task-owned dependencies. Task 6 is `verified locally`
and `candidate-only`; atomic field/outbox persistence, staged attachment bytes,
sync, production deployment, and production evidence remain `not run` or
`blocked`.

Remaining application slices must add their authorized field-storage,
attachment, sync, route, security, migration/restore, artifact, responsive,
accessibility, and task-owned process/container cleanup gates before those
capabilities can be reported as `verified locally`. Remote CI remains separately
authorized.

For first-party identity or canonical local-preprod changes, run:

```bash
go -C ../../shared/auth test -count=1 ./...
go -C apps/api test -count=1 ./internal/identity ./internal/platform/session ./cmd/preprod-canonical-demo-identity-loader
node --test tests/local-compose-policy.test.mjs tests/preprod-data-boundary.test.mjs
docker compose --file deploy/local/compose.yaml config --quiet
./scripts/test-canonical-preprod-fault-restart.sh
git diff --check
```

Expected: public OIDC and private administration remain split; the gateway
cannot reach private administration; nine fresh synthetic subjects reconcile
exact provider/application authority; HTTPS/HTTP role, lifecycle,
dependency-loss/restart, and public-admin denial checks complete; and
task-owned residue is zero. This remains `candidate-only`; release is
`release pending`.

## Governed AGA Intake And Official-Source Authoring Lane

Use this lane for the approved AGA checklist intake/authoring plan. It is
boundary-sensitive and remains local `candidate-only` evidence.

```bash
node scripts/verify-governed-checklist-test-inventory.mjs --phase gate0
AGA_CHECKLIST_ARCHIVE='/path/to/AGA - Checklists and Form.zip' \
  node --test tests/governed-checklist-intake-plan-contract.test.mjs \
  tests/governed-checklist-intake-security.test.mjs \
  tests/aga-checklist-archive-inventory.test.mjs
./scripts/test-governed-checklist-intake-profile.sh --security-only
node scripts/verify-governed-checklist-test-inventory.mjs --phase task8
node scripts/verify-governed-checklist-test-inventory.mjs --phase final
node scripts/check-governed-checklist-intake-cleanup.mjs
```

The focused OpenAPI/generated, Go, React/HTTP, typecheck/build, artifact, root
legacy, and harness commands from the active plan are required for the slices
they own. The archive verifier streams/hash-checks only the explicit external
path and writes nothing. A successful `gate0`/`task8` exit is `verified locally`;
`task9` and `final` may exit `2` only for the explicit real Form 048 mechanism
and Phase 2 expansion authorization blocker after all required artifacts and
Vitest/Playwright runner discovery are non-zero and present. Missing artifacts
remain failures, never skips. Live PostgreSQL, MinIO, browser,
source-owner, reviewed-source-set, assignment-provisioning, Department Manager,
release, and production evidence are `blocked` until separately authorized.

## Current Harness Completion Gate

For agent harness readiness work, run:

```bash
git diff --check
node tests/harness-docs-smoke.test.js
node tests/demo-boundary-smoke.test.js
rg -n "docs/agent-harness|agent-harness/index|output-contract|verification-matrix|entropy-cleanup" AGENTS.md MANIFEST.md docs
if [ -d .github ]; then find .github -maxdepth 3 -type f; fi
```

Expected `.github` result for this task: no new workflow file. If an unrelated
pre-existing `.github` file appears, do not delete it; report the ambiguity.
If a task-owned workflow appears, remove it before reporting success.
