# Demo Record Presentation

Date: 2026-08-17
Last updated: 2026-08-17
Status: active — implementation started; `candidate-only`; `release pending`

## Objective and user-visible outcome

Remove opaque aggregate and command identifiers from the Department Manager
demo's visible record labels. Planning, Audit, Potential Finding, Finding, and
CAP surfaces will lead with the business title and use a short stable reference
only where distinguishing identical records is necessary.

## Scope and exclusions

- In scope: React demo labels, button names, select options, queue/table cells,
  dossier headings, and focused regression tests.
- Technical IDs remain unchanged in backend calls, routes, data attributes, and
  persistence; this plan does not alter lifecycle semantics or data.
- No backend, external environment, deployment, or production data changes.

## Ordered work and verification

1. Add a shared presentation helper for short, stable record references.
2. Apply it to the Department Manager Planning, Audits, Findings, and CAP
   pages and the Lead Potential Finding queue.
3. Add focused regression tests, run typecheck, focused tests, demo build, and
   a local browser visual check at desktop and narrow widths.
4. Record literal outcomes here and update this plan/index only after
   inspection.

## Current progress and recovery

- `verified locally`: source, screen captures, and repository authorities read.
- `not run`: implementation verification and browser visual check.
- Recovery is a source-only revert of the changed presentation files; backend
  identity and persisted data are untouched.

## Execution Prompt

Continue in `apps/surveil`. Preserve unrelated dirty work. Complete the
visible record-presentation remediation, run the named local checks, and
record literal evidence without claiming release or production readiness.
