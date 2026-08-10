# Canonical AGA Manual Viewport Review — 2026-08-10

Status: `candidate-only`

Stakeholder disposition: `accepted` — 2026-08-11

This folder is the privacy-safe handoff for the manual Question Review and New
Audit viewport check. The images were captured from a task-owned disposable
local-preprod profile built from the current worktree. They are not release,
production-readiness, or external-preprod evidence.

## Review set

| Surface | 1440x900 | 1024x768 | 390x844 |
|---|---|---|---|
| New Audit exact subset | [open](new-audit-1440x900.png) | [open](new-audit-1024x768.png) | [open](new-audit-390x844.png) |
| Question Review queue and Decision file | [open](question-review-1440x900.png) | [open](question-review-1024x768.png) | [open](question-review-390x844.png) |
| Question Review history and decision controls | [open](question-review-decision-1440x900.png) | [open](question-review-decision-1024x768.png) | [open](question-review-decision-390x844.png) |

The machine-readable capture receipt, dimensions, scroll positions, byte
sizes, and SHA-256 digests are in [capture-manifest.json](capture-manifest.json).

## Automated capture facts

- nine of nine screenshots were written at the exact requested viewport sizes;
- New Audit selected three exact immutable question versions from the
  server-owned 1,310-question disposable exercise catalog;
- Question Review used the exact owned scope and `PREPROD_EXERCISE` boundary,
  recorded a real `RETAIN` exercise decision with the controlled
  `MANAGER_SCOPE_DECISION` reason, and projected its append-only Draft history;
- the queue/Decision-file relationship and the controlled decision,
  reclassification, topic, technical-disclosure, and publication-denial
  controls are visible separately at all three viewport sizes;
- document-level horizontal overflow was false at every viewport;
- product-page HTTP errors, browser console errors, uncaught page errors, and
  request failures were all recorded explicitly as zero;
- the connected authorization and queue response were verified before an
  invented privacy-safe presentation projection replaced visible catalog and
  dossier content; no real AGA question body or visible catalog label was
  retained in these screenshots or the capture receipt; and
- the isolated browser context was closed after capture.

The mobile-overflow defect found during preparation was fixed with a deliberate
RED CSS ownership test. Browser OTLP was also explicitly disabled only for the
canonical local-preprod build, which does not provision a collector; the
collector-backed full profile keeps telemetry enabled.

## Stakeholder checklist

Review the nine images and record either acceptance or concrete notes for:

- information hierarchy, density, readability, and control clarity;
- New Audit exact-question selection at desktop, tablet, and mobile widths;
- Question Review queue/Decision-file relationship and responsive stacking;
- absence of clipped text, unintended horizontal scrolling, or hidden primary
  actions; and
- whether the surfaces are suitable for the current local demo milestone.

The user accepted this manual review on 2026-08-11; the durable scope and claim
boundary are recorded in the
[stakeholder disposition](../stakeholder/CANONICAL_AGA_STAKEHOLDER_DISPOSITION_2026-08-11.md).
The separate external-preprod plan remains paused and `not run`.
