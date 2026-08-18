# Authorized Planning Scope Options

Date: 2026-08-18
Status: active — source and local mock verification complete; connected database verification `blocked`; public release `release pending`; production readiness not claimed

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
- `blocked`: integration qualification test because PostgreSQL at
  `127.0.0.1:55432` was not running.
- `release pending`: released demo lock still points at the prior three
  bootstrap manifest digests; it was intentionally not edited.

## Risks, idempotence, and recovery

Foundation and catalog writes remain idempotent and replay-checked. Exact
scope/target identity and Department Manager authority remain server-owned.
If connected bootstrap verification fails, stop before any release action and
retain the prior immutable lock and public runtime. Revert only this plan's
known source changes through a reviewed patch if the tuple contract is rejected.

## Current outcome

The source contract and local mock path now support and render three supplier
options, three provider options, three aerodrome targets, and the existing
Ramp/Cabin plus eight controlled lifecycle inspection types. The live public
demo cannot reflect the new data until a new application/bootstrap release is
explicitly authorized and qualified.

## Execution Prompt

Run the remaining verification for this plan from the AviaWorkspace root:

```text
Read the root and apps/surveil AGENTS.md files and this plan. Preserve all
unrelated dirty changes. Start only the disposable local PostgreSQL profile,
run the qualification bootstrap replay/drift test, and inspect the exact
scope-option counts through the HTTP route. Do not edit or regenerate a
released lock, deploy, push, or change credentials without explicit approval.
If the connected checks pass, prepare (but do not publish) the exact release
inputs and report the required lock/image/qualification gates as release
pending.
```
