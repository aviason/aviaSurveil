# AGA Hybrid Classification And Synthetic Demo Lifecycle — Task 8 Evidence

Date: 2026-08-04

Status: `verified locally`

Product status: `candidate-only`

Release: `release pending`

Production-ready: not established

## Implemented

- Added the inspection/checklist response page with server-pinned question
  snapshots, exact answer commands, Potential Finding eligibility, precise
  disabled reasons, BFCache/principal purge handling, CAA-only role history,
  and Auditee public-owner projection handling.
- Added the Lead Potential Finding return, dismiss, and explicit conversion
  page with severity, CAP, Evidence, and due-date choices.
- Added the Auditee/CAA CAP and Evidence page with append-only revision/version
  controls, separate Comment to Auditee and Internal CAA Note fields, Evidence
  verification outcomes, and separate authorized closure.
- Registered fixed, identifier-free lifecycle suffixes beneath the five
  capability-gated role routes. Added responsive lifecycle styling and kept
  Manager recommendation/release controls fail-closed until exact server-
  derived scope, target, Draft, taxonomy/run, readiness, and binding facts are
  returned.
- Added the three required preprod Playwright specs with trace, screenshot,
  and video disabled in the `preprod-aga-demo` project.

## Verification

Focused UI tests:

```text
npm --prefix apps/web test -- src/features/inspections/aga-demo-inspection-page.test.tsx src/features/findings/aga-demo-potential-finding-page.test.tsx src/features/caps/aga-demo-cap-evidence-page.test.tsx src/app/aga-demo-workspace-routes.test.tsx
Test Files  4 passed (4)
Tests       16 passed (16)
```

Additional local gates:

```text
npm --prefix apps/web run typecheck                                      passed
npm --prefix apps/web run build:demo                                     passed
assert-aga-workspace-artifact-boundary --profile demo                    passed
npm --prefix apps/web run build:http                                     passed
assert-aga-workspace-artifact-boundary --profile http                    passed
assert-http-artifact                                                      passed
```

The required discovery command passed with a nonzero result:

```text
npm --prefix apps/web run test:e2e:aga-preprod -- --list aga-hybrid-classification-workspace.http.spec.ts aga-synthetic-lifecycle.http.spec.ts aga-hybrid-privacy.http.spec.ts
Total: 7 tests in 3 files
```

The focused tests cover Inspector answer/proposal and role gating, Lead
return/dismiss/convert, Auditee CAP privacy, CAA-only role history, public
owner labeling, CAP-versus-closure semantics, precise disabled controls,
keyboard selection, narrow viewport behavior, fixed route suffixes, and
BFCache purge. No connected browser run was performed; browser execution is
`not run` until Task 9's isolated services and separately issued authority are
available. No production or real database claim is made.
