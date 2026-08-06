# AGA Demo Browser QA Review — 2026-08-06

## Scope and current result

`verified locally`

This is a fresh browser pass against the running local-preprod AGA demo at
`http://127.0.0.1:4174/department-manager/aga-demo-workspace`. The historical
connected lifecycle evidence in
`AGA_MANAGER_MULTI_ROLE_DEMO_2026-08-05.md` remains a separate run; this report
records the current browser state and current-instance regressions.

The demo is `candidate-only`, `release pending`, and
`production-ready: not established`. Read-only classification and bounded
package-preview flows are usable. A fresh Manager → Inspector → Lead → Auditee
synthetic lifecycle cannot currently be completed from the browser.

The local status check reported:

```text
AGA API demo responding
URL: http://127.0.0.1:4174/department-manager/aga-demo-workspace
API: http://127.0.0.1:58081
Questions: 1310
```

## Findings

### F-001 — Lifecycle pages cannot obtain a server-bound inspection

**Severity:** blocker for end-to-end demo readiness
**Evidence:** verified locally

On the Manager, Inspector, Lead Inspector, CAA Reviewer, and Auditee AGA
lifecycle routes, the browser rendered `Backend request failed with status
404`. The lifecycle context remained empty with the explicit message that no
server-returned synthetic inspection was available. `Load authorized
inspection`, `Start inspection`, finding decisions, and CAP/Evidence actions
were disabled with truthful reasons.

The package builder also reported `1272 current Draft leaves still need an
explicit INCLUDE, EXCLUDE, or DEFER disposition`, so a fresh demo start is not
release-ready and does not expose the intended multi-role lifecycle without
additional setup work.

### F-002 — Enabled classification include action returns a neutral 404

**Severity:** high for classification usability
**Evidence:** verified locally

On the Manager classification route, selecting `FSS-AGA-FORM-002 · 1` and
clicking `Include selected item` produced the visible alert
`Backend request failed with status 404`. The row remained `Disposition not
set` and the counters did not change.

The same item was shown by the package preview as one exact match with
`0 eligible / 1 ineligible`; the valid `EXCLUDE` batch path did succeed. The
server is therefore fail-closed, but the browser exposes an enabled action
without explaining the eligibility boundary and surfaces a generic 404.

### F-003 — Package preview starts with a form that has no candidates

**Severity:** medium
**Evidence:** verified locally

The package builder's default batch form is `FSS-AGA-FORM-001`. Searching the
sealed inventory for that form returned zero rows and the first server preview
returned `0 exact matches`. The actual inventory begins with
`FSS-AGA-FORM-002`; the user must discover and change the form filter before a
useful preview is possible.

### F-004 — Visible logout does not clear the identity-provider SSO

**Severity:** high for multi-role demo handoff
**Evidence:** verified locally

After logging out through the visible browser control, starting another OIDC
login reused the previous Keycloak identity. Attempting to switch from the
Manager to the Inspector consequently returned `Not available for this role`
instead of presenting the requested account. The qualification helper avoids
this by explicitly clearing the browser cookie jar; the visible UI does not.

### F-005 — Reloading the package page reuses idempotency keys

**Severity:** medium
**Evidence:** verified locally

The package page keeps its command sequence in component state. After a page
navigation/reload, the sequence starts again at one while the server retains
the previous operation keys. Subsequent previews returned `Workspace command
conflict` until the client advanced to an unused per-operation key.

### F-006 — Inspector landing screen surfaces a non-JSON 404

**Severity:** low/medium
**Evidence:** verified locally

After a successful Inspector OIDC login, the default
`/inspector/inspector-assignments` screen displayed
`Backend response 404 did not use a JSON content type.` The AGA handoff route
loaded its own lifecycle guard, but the role landing screen still exposed a
backend error to the user.

## Positive checks

- OIDC login and exact return-to behavior worked for Manager, Inspector, Lead
  Inspector, Auditee, and Admin accounts.
- The 1,310-row sealed inventory loaded; search, confidence filtering, and
  pagination responded without console warnings or errors.
- A server-issued package preview and a valid atomic `EXCLUDE` disposition
  completed with the status `Confirmed simulation disposition appended as one
  atomic Draft successor.`
- Unauthorized role routes rendered `Not available for this role` without
  exposing another role's projection.
- The Auditee CAP/Evidence screen did not expose `Internal CAA Note` or private
  risk fields.
- Technical approval, publication, and readiness controls were disabled with
  truthful candidate-boundary reasons.
- Desktop layout at 1280×720 had no horizontal overflow.

## Focused checks

```text
npm --prefix apps/web test -- \
  src/features/checklists/aga-classification-workspace-page.test.tsx \
  src/features/inspections/aga-demo-inspection-package-page.test.tsx \
  src/app/aga-demo-workspace-routes.test.tsx
3 files, 15 tests passed

npm --prefix apps/web test -- \
  src/backend/aga-demo-workspace.test.ts \
  src/backend/http-backend.test.ts
2 files, 25 tests passed

git diff --check
passed
```

## Disposable test-state note

The browser pass intentionally executed one valid batch `EXCLUDE` operation to
prove the write path. The disposable Draft was left at `37 included / 1
excluded / 0 deferred`; no canonical record was changed. A destructive
generation reset was not run.

## Readiness decision

The current instance is **not end-to-end ready**. It is suitable for
read-only classification exploration and bounded package-preview testing, but
not for a one-link, multi-role lifecycle demonstration. Readiness requires a
server-returned inspection after completing the Draft, a user-facing fix for
the enabled-but-neutral classification action, full identity-provider logout
for role handoff, stable idempotency keys across reloads, and a valid default
package filter. Production, deployment, external identity, legal/source-owner
attestation, and real-device claims remain `not run`.

## Independent GPT-5.6-sol ultra second pass

blocked for delegated browser reproduction: the isolated ultra runtime had no
available in-app Browser and 127.0.0.1:4174 refused connections. This does not
replace the main browser evidence above. The ultra pass therefore used a
read-only source, API, and focused-test audit; it made no edits and issued no
mutating requests.

The audit corroborated F-001 through F-006, including the lifecycle neutral
404, the classification Include failure, the empty default form, IdP SSO
reuse, component-local idempotency collisions, and the Inspector landing
error. It also found these additional issues:

### F-007 — Same-organization lifecycle access is not aggregate-scoped

**Severity:** high security and authority risk
**Evidence:** source audit; not independently reproduced in the browser

Lifecycle visibility accepts an authorized same-organization CAA principal,
but the binding-pin check does not compare the aggregate's pinned subject,
department, or unit with the current principal. CAP review and authorized
closure similarly check role and organization without binding the operation to
the aggregate scope. A wrong-unit same-organization manager or reviewer could
therefore read CAA history/internal notes and mutate another unit's aggregate
when holding valid current CAS values. Relevant source locations are
apps/api/internal/agademoworkspace/lifecycle_projection.go, lifecycle.go, and
authorization.go.

### F-008 — Batch confirmation can execute stale hidden preview values

**Severity:** high for disposition integrity
**Evidence:** source audit; not independently reproduced in the browser

After a package preview is created, changing the visible filter, action, or
reason does not invalidate the stored preview. Confirm submits the previous
preview's hidden values while the visible summary omits those values. A user
can consequently believe the current controls apply while up to 500 rows
receive the earlier action.

### F-009 — Base-row classification commands use the wrong key namespace

**Severity:** high for classification mutation integrity
**Evidence:** source audit; corroborates F-002

The base-row UI sends a bare BaseIdentity.Key(), while Draft lookup requires
the QuestionRef.Key() namespace prefixed with base\x1f. This affects Retain,
Include, Exclude, Defer, reclassification, topic, proposal, and reword
actions. Batch disposition constructs the prefixed key correctly, which
explains why the observed row-level Include returned a neutral 404 while the
valid batch EXCLUDE path worked.

### F-010 — Draft/lifecycle mutation and idempotency receipt are not one atomic commit

**Severity:** high for retry and audit integrity
**Evidence:** source and plan audit; not independently reproduced in the browser

The mutation is committed through one SQL function call and the idempotency
response is stored separately. A crash between those commits can leave a
durable mutation without its replay receipt; a retry may then conflict or
duplicate instead of returning the committed response. Direct Draft CAS is
also checked outside the database transaction, so concurrent commands can race
into a revision-key failure. This diverges from the active plan's
one-transaction requirement.

### F-011 — Preview expiry makes deterministic previews unrecoverable

**Severity:** medium
**Evidence:** source audit; not independently reproduced in the browser

The preview ID excludes expiry while stored-record equality includes expiry.
Regenerating the same preview inputs after expiry can therefore conflict
instead of producing a fresh preview.

### F-012 — Initial Draft metadata can render base dispositions as UNSET

**Severity:** medium
**Evidence:** source audit; not independently reproduced in the browser

Initial metadata mapping repeats the bare-versus-prefixed base-key mismatch,
so persisted base-item dispositions may appear unset in the initial reader
projection even when the stored Draft has a disposition.

### F-013 — Confirm remains enabled for guaranteed-failure previews

**Severity:** medium
**Evidence:** source audit; complements F-002/F-003

The package UI leaves Confirm available for zero-row previews and for Include
previews containing only ineligible rows. Both requests are guaranteed to be
rejected by the server instead of being prevented or explained before submit.

### F-014 — Package refresh can retain stale lifecycle artifacts after errors

**Severity:** medium
**Evidence:** source audit; not independently reproduced in the browser

Refresh handling swallows recommendation/current-inspection errors and can
retain stale release artifacts across reset, 404, ambiguity, or authorization
failure. The screen can therefore describe an earlier server state after the
current request has failed.

### F-015 — Admin capability and mutation authorization disagree

**Severity:** medium
**Evidence:** source audit; not independently reproduced in the browser

The server advertises AGA classification/recommendation/lifecycle capability
to Admin and the UI enables draft mutation controls for that role, while API
authorization and OpenAPI restrict those commands to Manager. Admin users can
therefore reach visible controls that are guaranteed to neutral-deny.

### F-016 — Binding resolution can compose roles across different scopes

**Severity:** medium security and authority risk
**Evidence:** source audit; not independently reproduced in the browser

Production binding resolution unions roles from all active same-subject,
same-organization bindings under the first binding's department/unit identity
without verifying scope equality. Different-scope bindings can consequently
become one synthetic authority for authorization and idempotency.

### F-017 — Unknown AGA routes fail open to misleading screens or logout

**Severity:** medium navigation and session-integrity risk
**Evidence:** source audit; not independently reproduced in the browser

An unrecognized one-segment AGA child silently renders classification. Other
unknown routes redirect to /, which mounts the authenticated logout route and
revokes the session. A malformed or stale deep link can therefore show the
wrong screen or unexpectedly sign the user out instead of returning a not-found
state.

## Ultra-pass verification and coverage caveat

The delegated source/test pass reported 27/27 focused web tests, the relevant
Go packages, and git diff --check passing. Those tests use permissive or
fabricated mocks in the affected paths and do not cover the failure modes
above; they are supporting evidence, not a release-readiness waiver.

## Merged readiness decision

The combined evidence remains **not end-to-end ready** and release pending.
The current instance is useful for read-only classification exploration and
bounded package-preview testing, but it is not suitable for a one-link,
multi-role lifecycle demonstration or an authority-sensitive production
workflow. The six browser findings require the runtime/demo fixes listed above;
the ultra pass additionally requires aggregate-scope authorization, corrected
question-key handling, stale-preview invalidation, transactional
mutation/idempotency storage, and capability/route consistency before any
production-ready claim.

## Independent GPT-5.6-sol xhigh third pass

The xhigh pass independently corroborated F-001 through F-017. It did not run
the Browser and made no edits or mutating API calls; these findings are
source-only or deterministically reproducible from the cited paths.

### F-018 — Auditee projection exposes Potential Findings and CAA workflow surfaces

**Severity:** high privacy and projection-boundary risk
**Evidence:** source audit; not browser-reproduced in this pass

The lifecycle projection includes Potential Findings in the Auditee response,
and the Auditee routes expose queues, states, comments/history, and CAA
review/closure panels. This conflicts with the Auditee portal contract and the
active lifecycle plan, which require the pre-conversion projection to exclude
CAA workflow material. Source locations: lifecycle_projection.go:245-279,303-324;
aga-demo-workspace-routes.tsx:36-45; aga-demo-potential-finding-page.tsx:95-128;
aga-demo-cap-evidence-page.tsx:115-136; AUDITEE_PORTAL.md:79-93.

### F-019 — Required reopen and authorized-closure reasons are discarded

**Severity:** high audit-integrity risk
**Evidence:** source audit; not browser-reproduced in this pass

The lifecycle handlers validate reason inputs but do not persist them, and the
event contract has no reason field. The UI supplies generic reason codes.
Required reopen and authorized-closure explanations therefore cannot be
recovered from the lifecycle history. Source locations:
lifecycle.go:301-305,575-585,629-672; contract.go:346-358;
aga-demo-inspection-page.tsx:449-454; aga-demo-cap-evidence-page.tsx:133-136.

### F-020 — Returning to a cached classification page loses sealed question text

**Severity:** medium usability and record-context risk
**Evidence:** deterministic source repro; not browser-reproduced in this pass

The classification page caches metadata without the question text and restores
that incomplete object as the active response. Navigating Page 1 → Next →
Previous consequently renders the missing-text fallback instead of the sealed
question content. Source locations:
aga-classification-workspace-page.tsx:71-81,177-199,403-406.

### F-021 — Lifecycle detail IDs are ignored or incompletely validated

**Severity:** medium/high exact-record authorization risk
**Evidence:** source audit; not browser-reproduced in this pass

An Auditee query with arbitrary non-empty Finding, CAP, or Evidence IDs can
still return the full inspection projection. CAA CAP/Evidence reads can also
ignore a bogus CAP ID when Evidence ID is absent. This violates exact-record
binding and neutral-denial semantics. Source locations:
lifecycle_projection.go:24-28,56-63,84-99.

### F-022 — Scoped Department Manager cannot receive lifecycle question text

**Severity:** high workflow and scope-resolution risk
**Evidence:** source audit; not browser-reproduced in this pass

Bindings use the organization alias CAA while the synthetic scope/aggregate
uses AGA-DEMO-CAA. Lifecycle projection requires literal equality, so a scoped
Department Manager can be denied question text even when the active plan
requires that role to receive it. Source locations:
postgres_store.go:474-540; lifecycle.go:84-89;
lifecycle_projection.go:133-145; active plan lines 304-309 and 642-646.

### F-023 — Very large page values can overflow into a query panic

**Severity:** high availability risk
**Evidence:** deterministic source repro; not browser-reproduced in this pass

Page values have no upper bound. The query handlers multiply page by page size
before checking bounds; a sufficiently large valid page value wraps negative on
64-bit and reaches a negative slice bound. This should return a bounded client
error instead of panicking. Source locations:
types.go:314-321; service.go:145-160;
lifecycle_projection.go:147-165. For example, page
368934881474191033 with page size 25 wraps negative on 64-bit.

### F-024 — Admin reset creates an incomplete generation

**Severity:** high authorization and reset-integrity risk
**Evidence:** source audit; not browser-reproduced in this pass

Admin reset creates the generation, Draft, and seal but omits provider scopes,
targets, and authority bindings. Current-generation scope lookup then fails,
while binding resolution has no current-generation predicate and can remain
tied to stale generations. Source locations:
postgres_provision.go:742-764; postgres_fact_resolvers.go:77-101;
postgres_binding_resolver.go:27-75; postgres_provision.go:290-295.

## Xhigh verification and merged status

The xhigh pass reported 8 focused web files with 53 tests passing, the relevant
Go packages passing with an isolated GOCACHE, and git diff --check passing.
Browser execution was not run in that pass. The combined report now contains
24 findings: six browser-verified findings, eleven ultra source/test findings,
and seven xhigh source/deterministic findings.

The merged readiness decision is unchanged: candidate-only, release pending,
not end-to-end ready, and production-ready not established. No code, commit, or
push was performed; only this evidence report and its docs index entry are
modified.
