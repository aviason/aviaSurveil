# Authorized Planning Scope Options

Date: 2026-08-18
Status: active — source, mock, disposable PostgreSQL qualification, immutable release, bootstrap, apply, and public health/app-shell verification `verified`; production readiness not claimed

## Objective and user-visible outcome

Make the Department Manager New Inspection cascade show three or more
server-authorized options where the selected branch supports them. The
existing canonical qualification tuple remains available, while the
local/demo data exposes multiple supplier organizations, provider scopes,
regulated targets, and the controlled inspection lifecycle types without
fabricating client-side choices.

## Scope and exclusions

Included:

- Curated foundation manifest support for additional organizations, scopes,
  target bindings, and owned facility/location targets.
- Roster support for more than one declared Department Manager authority so
  every added provider scope remains subject to server-side authorization.
- Approved catalog applicability for multiple exact scope/target bindings.
- Test/mock scope options and planning-cascade coverage.

Excluded:

- Public deployment, image rebuild, release-lock mutation, credential changes,
  or external infrastructure actions.
- Any frontend fallback that invents options when the server returns none.
- Real aviation organization data or a change to regulatory meaning.

## Ownership and affected interfaces

AviaSurveil owns the React mock contract, Go bootstrap loaders, and HTTP
server behavior. AviaWorkspace owns deployment manifests and release locks.
The affected interfaces are:

- `FoundationManifest` and its curated foundation loader;
- `RosterAccount` Department Manager authority declarations;
- approved catalog scope-binding import/replay verification;
- `CanonicalCatalogBackend.listScopeOptions` mock behavior;
- New Inspection's existing server-owned cascade.

## Ordered work and verification

1. Extend manifest and loader contracts for additional authorized tuples.
2. Add demo/dev curated tuples and matching approved-catalog applicability.
3. Add mock options and an assertion covering supplier, provider, target, and
   inspection-type cascades.
4. Run focused Go and React tests, both web builds, manifest validation, and
   local browser DOM/screenshot checks.
5. Run the disposable PostgreSQL qualification bootstrap once the local DB is
   available; then create a separately authorized immutable release candidate.

Fresh evidence from this change:

- `verified locally`: Go loader packages, 20 React planning tests, `build:demo`,
  `build:http`, foundation manifest validation, approved catalog validation,
  and local IAB interaction proof.
- `verified locally`: `bash scripts/test-qualification-bootstrap.sh` passed the
  fresh PostgreSQL foundation/roster/catalog replay, including expansion from
  one historical applicability binding to the current authorized binding set.
- `verified`: the separately authorized `namibia/demo` successor release is
  live under immutable lock `sha256:643b4b11722687ee61daf8e5cd763ab59159cbbab07a65389129d0eaf0a39135`.
- `verified`: public `/health/ready` returned 200 and the public app-shell
  fingerprint, worker digest, and manifest digest matched the lock; the
  public `/auth/session` 401 remains expected without a browser session.

## Risks, idempotence, and recovery

Foundation and catalog writes remain idempotent and replay-checked. Exact
scope/target identity and Department Manager authority remain server-owned.
If connected bootstrap verification fails, stop before any release action and
retain the prior immutable lock and public runtime. Revert only this plan's
known source changes through a reviewed patch if the tuple contract is rejected.

## Current outcome

The source contract and local mock path support and render three supplier
options, three provider options, three aerodrome targets, and the existing
Ramp/Cabin plus eight controlled lifecycle inspection types. The live public
demo now reflects the authorized foundation/catalog/roster release. Future
recommendation-policy or catalog changes require their own plan and immutable
release gate.

## Execution Prompt

The implementation and separately authorized public release gates for this
scope-options plan are complete. Do not mutate the live demo from this plan.
Future work should use a new ExecPlan and an exact release authorization.

```text
Read the root and apps/surveil AGENTS.md files and the next active plan. Keep
the immutable demo release unchanged, preserve unrelated dirty changes, and
obtain separate authorization before any new image, bootstrap, lock, or cloud
operation.
```
