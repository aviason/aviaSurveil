# App-Shell Cache And Exact-Vector Convergence

Date: 2026-08-17
Last updated: 2026-08-17 (P0-5 source handoff recorded)
Status: active — exact `namibia/demo` release deployed; public worker/manifest/health and no-op verified; retained pre-monitor browser convergence remains browser-scheduled

## Planning authority

This plan is governed by [`docs/PLANS.md`](../../PLANS.md). The root
cross-repository plan owns Workspace composition and release boundaries. The
Auth child plan owns provider transaction and authorization-code security.

## Objective and user-visible outcome

Generate a content-bound app-shell fingerprint and complete file manifest,
activate only exact-vector-compatible Service Workers, preserve durable offline
data, keep gateway-owned routes network-only, and retire the legacy v9 worker
and document at cutover without reviving any OIDC artifact.

The safe retirement bridge waits for a quiescent, durable client checkpoint
before takeover and requests a user-controlled reload after the exact
successor activates. It retains durable IndexedDB/OPFS/outbox state and never
forces navigation while local work may exist.

Visible online clients explicitly check the stable worker at startup, every 60
seconds, on `pageshow`, after reconnecting, and after returning to the
foreground. The gateway serves the current stable worker body from the legacy
`/sw.js?v=9` registration URL so clients trapped behind a cached legacy
document can enter the same verified activation path.

## Scope and ownership

- Surveil owns the browser, Service Worker, web server, local Caddy, and local
  browser/runtime tests.
- Workspace owns cloud Caddy, release artifacts, exact Terraform action gates,
  and optional separately authorized Cloudflare edge operations.
- Auth owns registered client restart metadata, stale Auth stage handling, and
  atomic authorization-code claim/finalization.
- Surveil BFF state generation and consumption remain unchanged except for
  focused regression tests.

The Offline-First Browser Production Hardening plan is the current authority
for the handed-off P0-5 safe checkpoint, unresponsive-client fence, and
no-forced-navigation behavior. This plan retains app-shell fingerprint,
manifest, gateway/cache, release provenance, and public cutover ownership.

No release lock, image, secret, Cloudflare, Terraform, deployment, or public
qualification state is changed by this implementation slice.

## Current progress

- `verified locally`: source and planning authorities reread.
- `verified locally`: dependency-free compatibility vector added with the
  current `{9,2,1,1}` values.
- `verified locally`: focused app-shell/update/quiescence tests, web typecheck, demo/http builds, Node/Python artifact verification, web-server cache tests, A/B/C fingerprint harness, local Compose health/residue checks, live local header matrix, and Caddy validation.
- `verified locally`: continuous update checks at startup, at a bounded
  60-second interval, on `pageshow`, on reconnect, and on foreground return;
  concurrent checks coalesce and transient failures retry.
- `verified locally`: the legacy `/sw.js?v=9` gateway route rewrites to and
  proxies the current stable worker instead of returning `410`.
- `verified locally`: a fingerprint-bound exact-vector legacy v9 predecessor
  can activate without a v2 cache marker, and exact-vector clients may skip a
  missed intermediate shell release.
- `verified locally`: a successor remains bound to the exact current v2
  predecessor while a separately embedded exact legacy v9 descriptor admits
  retained pre-v2 registrations that have no committed v2 cache marker.
- `verified locally`: client navigation is scheduled without awaiting it from
  the `activate` event, removing the activation/navigation deadlock; verified
  CacheStorage state restores the manifest after a worker-process restart.
- `verified locally`: the isolated persistent-browser test installed a legacy
  worker and cache, promoted the server to the current artifact, upgraded two
  open clients, moved the registration to `/sw.js`, deleted the legacy cache,
  stopped the worker process, reloaded offline from the restored verified
  cache, and preserved the local sentinel (`1 passed`).
- `verified locally`: focused app-shell tests passed `52/52`, typecheck,
  demo/http builds, artifact scans, A/B/C harness, focused Caddy contract, and
  Caddy native validation passed.
- `verified locally`: the P0-5 handoff slice now requires explicit
  safe-checkpoint ACKs with zero quiescence counters and durable-work
  acknowledgement; worker takeover does not force navigation.
- `verified live diagnosis`: `https://demo.aviasurveil.com/` serves the current
  HTML; after release, `/sw.js?v=9` and `/sw.js` both return HTTP 200,
  `no-store`, identical bytes, and worker SHA-256
  `fa03db49f7e18df60fed488cd5559d387fe8030eb4dee7fbff8c3982985ea762`.
- `verified`: public manifest SHA-256 is
  `cd86cd35aba7d6169eef81283c4f63d853fe9d7094b29158cc3a7bc304901c76`,
  release fingerprint is `sha256:e933f77a596f06e06969a1115c93fc7b27bfcf3c656f2f6a80895e1b606664a1`,
  and public HTML names `app-BTQeTkmh.js` and
  `workspace-shell-D69tKOkH.js`.
- `verified`: exact `namibia/demo` lock
  `sha256:82729d03896ebfd8908fd5ed202178788202126385b7aca9c8cc75b0c6016ad9`
  was applied through the seven-action saved plan; public readiness and
  target-specific smoke passed, then detailed-exitcode reported `No changes`.
- `blocked`: a retained browser profile still running a pre-monitor cached
  document did not immediately converge on reload. The server cannot invoke
  `registration.update()` inside that already-cached document without browser
  cooperation and site-data clearing was not performed. Its legacy worker URL
  now serves the exact bridge; all clients that reach this release have
  bounded automatic checks for future releases.
- `blocked`: the fresh full web suite ended at `682 passed / 8 failed`;
  failures are in unrelated dirty planning/management presentation surfaces,
  not the focused app-shell tests.
- `verified locally`: the Auth child migration/code-claim gate passed with no skipped required PostgreSQL tests.
- `blocked`: the in-app Browser rejected the local Caddy internal CA; clean HTTP preview browser coverage passed for root, lazy Inspector route, and mobile route.
- `not run`: Playwright WebKit because the local WebKit binary is unavailable;
  the same persistent-browser test supports
  `AVIA_LEGACY_UPDATE_BROWSER=webkit` when that binary is present.
- `not run`: destructive site-data clearing, broad Cloudflare purge, public
  mutating qualification, preprod, and prod.
- `verified locally`: the completed app-shell implementation slice is handed
  to the Offline-First Browser Production Hardening plan for its P0-5 safe
  checkpoint and unresponsive-client fence work. This handoff covers
  `apps/web/src/sw.ts`, `apps/web/src/offline/update-coordinator.ts`,
  `apps/web/src/offline/client-quiescence.ts`, and their focused tests. The
  app-shell plan retains release provenance, gateway/cache ownership, and
  public cutover authority.

## Ordered implementation

1. Finish the dependency-free vector, strict manifest/fingerprint generation,
   candidate CacheStorage validation, positive route policy, and strict
   retirement protocol.
2. Add Auth code claim/finalization before stale-login recovery. Do not ship
   stale recovery if the concurrent one-shot test is unavailable or fails.
3. Add web-server/Caddy cache headers and explicit `/http-config.json`
   no-store fetching.
4. Add isolated A/B/C browser tests using task-unique project/state/ports and
   verify forced legacy retirement plus predecessor-cache deletion.
5. Run native Surveil and Auth source gates; report missing fixtures literally.
6. Observe the retained Safari profile after its browser-scheduled worker
   update check; future releases use the deployed stable-worker monitor.

## Verification matrix

```bash
npm --prefix apps/web test
npm --prefix apps/web run typecheck
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
npm --prefix apps/web run check:app-shell
node --test tests/*.test.js
bash scripts/check-compose-policy.sh
git diff --check
```

After the task-owned cache harness exists, run:

```bash
bash scripts/test-cache-update-harness.sh
AVIA_E2E_PROFILE=offline npm exec --prefix apps/web -- playwright test tests/e2e/offline-legacy-worker-upgrade.spec.ts --project=offline
```

The harness must prove A/B/C manifest determinism, per-file hash validation,
failed candidate cleanup, forced legacy-v9 retirement, predecessor-cache
deletion, Chromium persistence, and WebKit history behavior.

## Acceptance criteria

- No browser/worker path clears IndexedDB, OPFS, outbox, packages, or
  attachment manifests.
- Automatic activation requires exact complete-vector equality.
- Legacy v9 clients remain waiting until the safe checkpoint is ACKed; after
  activation they receive a user-controlled reload request and are never
  force-navigated while local work may exist.
- A visible online client checks for a successor within 60 seconds; foreground,
  reconnect, startup, and page-show transitions check immediately.
- `/sw.js?v=9` returns the current stable worker body with `no-store`; it must
  not return `410` while a retained legacy registration can still exist.
- The `activate` event never awaits a client navigation.
- `/api`, `/v1`, `/auth`, `/identity`, `/health`, `/operations`, `/otel`,
  private routes, and `/http-config.json` remain network-only.
- Every committed app-shell response is validated for origin, redirect status,
  media type, byte size, and SHA-256.
- All required gates use existing commands or commands added by this plan;
  `scripts/test-http-profile.sh` is not referenced.

## Recovery and release boundary

Failed local candidate installation deletes only its candidate cache. A vector
change is a separate migration plan. No commit, push, image publication,
Terraform action, Cloudflare change, or public transition was performed in the
current implementation turn; those actions require explicit current-task
authorization. No broad Terraform action or ad-hoc AWS infrastructure mutation
is allowed.

## Execution Prompt

Continue from this plan in the Surveil checkout. Preserve unrelated changes.
Re-run the focused gates, review only task-owned changes, then obtain explicit
authorization before commit, push, image publication, or the exact
`namibia/demo` release flow. After release, require `/sw.js?v=9` and `/sw.js` to
return the same current worker body before retained-profile Safari validation.
