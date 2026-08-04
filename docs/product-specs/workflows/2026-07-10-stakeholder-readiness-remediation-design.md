# Stakeholder Readiness Remediation Design

**Date:** 2026-07-10
**Status:** approved
**Source review:** Independent final quality review of `b1782c2^..961d5c2`

## Objective

Resolve the independent review's blocking and material suggestion findings so
the AviaSurveil360 frontend-only demo can truthfully return to
`ready-for-verification` and be presented for stakeholder review. The work must
restore one report lifecycle, explicit role authority, Service Provider privacy,
state-backed report workflows, usable responsive layouts, and synchronized
English/Turkish product truth.

## Chosen Approach

Use a contract-first remediation sequence. Repair shared state and authority
contracts before changing role-specific rendering, then repair responsive and
interaction behavior, synchronize documentation, and finish with an independent
rendered verification gate.

This sequence is preferred over UI-first patching because the current visible
defects are downstream of report identity, ownership, and privacy errors. A broad
framework or architecture rewrite is also rejected because the repository is a
static Vanilla JavaScript demo and the required corrections fit its existing
module boundaries.

## Architecture And Boundaries

### 1. Canonical report lifecycle

- `auditReports` remains the only mutable report approval package collection.
- `managerReports` remains a read/list projection and must not hold an
  independently mutable approval state.
- Preliminary and Final Reports must resolve to distinct canonical artifacts;
  `PR-2026-018` and `FR-2026-018` must not resolve to the same Preliminary
  package.
- Exact helpers consume `reportId`, `auditId`, and `reportType`; first-match by
  audit is not permitted in interactive decision paths.
- Department Manager, GM, ED, Lead Inspector, and Service Provider views must
  observe the same report status, owner, lock state, timestamps, and history.

### 2. Authority and closure rules

- GM is an intermediate reviewer. GM may return a Final Report or forward it to
  ED, but may not issue, sign, lock, release, or claim final authorization.
- ED is the only demo Final Report issue/mock-sign/lock authority.
- Decision helpers validate both record eligibility and actor role; UI routing
  alone is not an authority boundary.
- ED report approval never closes an open Finding. An audit remains
  `Follow-up Open` while required CAP, Evidence, verification, or authorized
  closure work is incomplete.
- CAP acceptance advances the Finding to Evidence-required work and never closes
  it.

### 3. Service Provider privacy

- Every auditee-visible notification, message, report, CAP, evidence item, and
  document carries and enforces `organizationId`.
- Messages must never relabel an external organization's notification as Fly
  Namibia.
- Service Provider Settings must not render Inspector workload, internal risk
  scoring, Oversight Health Index internals, enforcement deliberations, or
  Internal CAA Notes.
- The role allowlist may keep working utility routes only when their content is
  auditee-safe; otherwise the route is removed or redirected to the Service
  Provider home.

### 4. State-backed operational surfaces

- Lead Inspector Final Reports use canonical Final Report rows and propagate the
  selected `reportId` through prepare, preview, attachment, record, and submit
  actions.
- No visible action may be inert. Each control must navigate, update visible
  state, open a modal, create the intended mock file, or be explicitly disabled.
- Multi-Inspector assignment uses the same question IDs as Cabin execution.
  Inspector execution and user-specific notifications consume the current audit,
  assignment, and Inspector user ID.
- Department Preliminary decisions, notifications, and audit-log entries use the
  selected Report ID instead of `PR-2026-018` literals.

### 5. Responsive and accessible interaction

- Final Report preview must have no page-level or nested horizontal overflow at
  `1536x864`, `1366x768`, `1024x768`, and `390x844` at 100% zoom.
- Mobile layout stacks queue, selection summary, detail, preview, and decision in
  task order. Primary actions remain visible and keyboard reachable.
- Report content may scale or reflow for screen preview while print/PDF styling
  retains the controlled page layout.
- Console errors, stale IDs, clipped status/action text, broken navigation,
  missing focus return, and inactive controls are release blockers.

### 6. Product truth and tracking

- Update canonical English docs and matching `.turkce.md` companions together.
- Correct the Inspector BUILD_SUMMARY from stale SMS/SkyCargo copy to the Fly
  Namibia Cabin Inspection behavior.
- Expand the canonical permission matrix to Finance, GM, and ED, including
  explicit GM-intermediate and ED-final authority.
- Reconcile `MANIFEST.md` with the actual discovered smoke-test inventory.
- Record the older governance plan's automatic audit-closure statement as
  superseded by the current Finding/CAP/Evidence closure rule.
- Mark the reviewed plan `blocked` until this remediation passes; create one
  active follow-up implementation plan and one durable `note-open` tracker entry.

## Testing Strategy

Each implementation unit requires focused regression coverage and a broader
verification gate:

1. Add a focused contract or render test for the reproduced defect.
2. Implement the smallest scoped correction.
3. Run the focused test and neighboring role/workflow tests.
4. Run all JavaScript syntax checks and `node --test tests/*.test.js`.

The final gate also requires a fresh isolated-browser matrix at all four target
viewports. It must exercise negative authority and privacy cases, selected-ID
continuity, every changed visible control, CAP/Evidence closure boundaries,
preview/PDF behavior, console output, page and nested overflow, and cleanup of
temporary browser/test-server processes.

## Error And Invalid-State Handling

- Unsupported actor roles return a visible validation result and do not mutate
  report state, history, notifications, or audit logs.
- Missing, mismatched, or stale Report IDs render a scoped empty/error state and
  never fall back to another audit's first report.
- Cross-organization Service Provider records are treated as unavailable and do
  not leak identifying text through empty states, counts, messages, or metadata.
- Controls without an available demo behavior are removed or visibly disabled
  with explanatory copy.

## Dependencies And Ownership

- Product owner: confirms stakeholder-facing role and authority wording.
- Engineering implementer: owns static state, helpers, views, actions, tests, and
  responsive CSS.
- Independent verifier: reruns the final evidence matrix and decides GO/NO-GO.
- Product/legal/security owners retain ownership of production identity,
  signature validity, authorization, enforcement, retention, and audit-log
  policy; the existing durable note remains open.

## Out Of Scope

- Backend, API, database, real authentication, or production authorization.
- Real e-signature, enforcement execution, audit-log service, notification
  delivery, file storage, or regulatory ingestion.
- Framework migration, design-system rewrite, deployment, branch creation,
  commit, push, or PR activity without separate current-task authorization.
- Production-readiness claims.

## Success Criteria

- Every Blocking finding from the independent review is covered by a failing
  regression test and a verified fix.
- Material Suggestions are either fixed or recorded as explicit accepted risks
  with owner and next action; none may be silently omitted.
- Tasks, docs, plan status, index next todo, and tracker state describe the same
  verified reality.
- The final independent readout is findings-first and returns GO with the literal
  labels `verified locally`, `not run`, `demo-only`, and
  `production-readiness not claimed`.
