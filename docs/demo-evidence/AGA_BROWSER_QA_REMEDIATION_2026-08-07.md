# AGA Browser QA Remediation Evidence — 2026-08-07

This is a candidate-only remediation record for
[the 2026-08-06 QA findings](AGA_BROWSER_QA_REVIEW_2026-08-06.md) and the
[active remediation plan](../exec-plans/active/2026-08-07-aga-browser-qa-remediation-plan.md).
It does not rewrite the original browser report or establish release or
production readiness.

## Current decision

Focused source, unit, contract, and boundary checks are `verified locally`.
The connected lifecycle, external OIDC end-session, database fault/concurrency,
and browser execution gates are `not run` or `blocked`. The result remains
`candidate-only`, `release pending`, and `production-ready: not established`.
The final independent GPT-5.6-sol xhigh acceptance is pending after the latest
source/UI remediation slice.

## Finding matrix

| Finding | Current evidence status | Remediation evidence / remaining boundary |
|---|---|---|
| F-001 | `not run` | Connected qualification and a current server-bound inspection were not executed. |
| F-002 | `verified locally` | Server emits Include eligibility/reason; UI and direct command fail closed for ineligible rows. |
| F-003 | `verified locally` | Server-first package form selection is covered by focused React tests. |
| F-004 | `blocked` | Local application-session logout is implemented; provider end-session is absent from the current contract. |
| F-005 | `verified locally` | Per-intent UUID command keys and reload-safe state tests pass. |
| F-006 | `verified locally` | AGA role landing avoids the unsupported Inspector assignment probe. |
| F-007 | `verified locally` | Subject-plus-department/unit/provider scope checks are enforced; no manager assignment pin exists in the aggregate model. |
| F-008 | `verified locally` | Preview values are frozen and invalidated when visible controls change. |
| F-009 | `verified locally` | Base commands use canonical `QuestionRef` keys. |
| F-010 | `blocked` | Domain mutations and idempotency receipts still use separate storage calls; universal transaction proof is absent. |
| F-011 | `verified locally` | Preview identity includes expiry and post-expiry intent receives a fresh identity. |
| F-012 | `verified locally` | Base metadata disposition mapping uses canonical keys. |
| F-013 | `verified locally` | Zero, over-cap, ineligible, and mixed Include confirmations are disabled. |
| F-014 | `verified locally` | Refresh failures clear stale recommendation/inspection state. |
| F-015 | `verified locally` | Admin has a distinct read/history/reset-only surface and no Manager mutation capability. |
| F-016 | `verified locally` | Operation-specific binding selection does not union roles across rows. |
| F-017 | `verified locally` | Unknown AGA suffixes render API-free not-found state. |
| F-018 | `verified locally` | Auditee transport and DOM use a positive CAP/Evidence/public-status allowlist; CAA review/verification/closure/history controls are absent. |
| F-019 | `verified locally` | Bounded lifecycle reason code and explanation are persisted in CAA audit projection only. |
| F-020 | `verified locally` | Sealed question text is refetched after page navigation. |
| F-021 | `verified locally` | Finding, CAP, Evidence, and relationship selectors validate before Auditee projection. |
| F-022 | `verified locally` | CAA/AGA-DEMO-CAA aliases use one server normalization path. |
| F-023 | `verified locally` | Page bounds are validated before multiplication/slicing. |
| F-024 | `verified locally` | Reset reconstructs a fresh fixture-bound generation with one ACTIVE publication; connected DB fault/concurrency proof is not run. |

## Fresh local checks

The latest source/UI slice passed:

- AGA Go packages and tagged preprod Go packages: `verified locally`.
- Focused React: 7 files, 43 tests: `verified locally`.
- OpenAPI generation/contract check: 16/16: `verified locally`.
- Boundary and demo smoke checks: 18/18: `verified locally`.
- Harness docs smoke and `git diff --check`: `verified locally`.

Full Vitest remains `not green` because the unrelated planning wizard file has
two reproducible failures; browser/Playwright execution and connected
qualification remain `not run`.
