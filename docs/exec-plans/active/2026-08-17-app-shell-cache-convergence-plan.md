# App-Shell Cache And Exact-Vector Convergence

Date: 2026-08-17
Last updated: 2026-08-19 (hands-free root release and retained Safari verified)
Status: active — exact `namibia/demo` hands-free successor release, public artifact/health/no-op, real Chrome worker/cache state, and retained Safari normal-root transition verified; authenticated workspace remains unverified

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
before normal takeover and permits a client-controlled automatic reload after
the exact successor activates. It retains durable IndexedDB/OPFS/outbox state
and never forces navigation while local work may exist.

Normal `demo.aviasurveil.com` entry remains the automatic update path. When no
broken worker already occupies `registration.waiting`, an exact-vector
successor promotes itself and quiescent legacy documents reload without a
separate URL.

The 2026-08-19 recovery correction also adds a stable network-only
`/app-shell-recovery.html` entrypoint. A client trapped behind an old cached
document can load this entrypoint from the network, replace an already waiting
broken candidate, and explicitly activate the verified exact-vector successor.
The worker does not call `clients.claim()`, navigate another client, or clear
origin storage. Quiescent legacy documents reload themselves; dirty legacy
documents remain open until their own work becomes quiescent.

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

The exact demo release may change only through the Workspace release lock and
allowlisted `namibia/demo` Terraform flow. Preprod, prod, secrets, DNS, and the
Cloudflare tunnel remain outside this slice.

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
- `superseded historical evidence`: the first bridge scheduled client
  navigation without awaiting it from `activate`. The 2026-08-19 correction
  removes worker-driven navigation; verified CacheStorage state still restores
  the manifest after worker-process restart.
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
- `verified live diagnosis`: on 2026-08-19 the retained Safari document named
  `app-BTQeTkmh.js`, `workspace-shell-D69tKOkH.js`, and
  `new-audit-wizard-B-ktUjlb.js`, while the public no-store HTML named the
  newer `app-BDwsof8P.js` and current release fingerprint
  `sha256:03cf09ace4b8a310bc230a24fa218b0ebd1195a0f6f9da8a25e299818566f533`.
  The deployed old client does not implement
  `avia:app-shell-safe-checkpoint-ack`, so the successor can remain waiting
  indefinitely; the 60-second update monitor alone cannot converge it.
- `superseded live finding`: the first public recovery release still required
  Safari `Reload Page From Origin` because the deployed legacy active worker
  intercepted both normal root and the recovery URL before either request
  reached the network. The recovery document worked once reached and cleared
  no site data, but it was not a hands-free path for that exact trapped state.
- `verified locally`: persistent Chromium regressions passed `2/2` with zero
  console/page errors. A normal root visit automatically upgraded an old
  stable-URL client when no worker was already waiting. The three-generation
  case then covered an intervening broken waiting worker and dirty legacy tab:
  the recovery entrypoint activated the repaired worker, preserved the dirty
  tab and local sentinel, then reloaded it after quiescence and retired legacy
  caches.
- `verified locally`: the hands-free correction promotes an exact-vector
  successor even when a broken worker already occupies `registration.waiting`,
  calls `clients.claim()` without worker-driven navigation, retains predecessor
  hashed assets/caches while a dirty legacy tab remains open, and lets each
  client reload itself only after quiescence. Focused tests passed `61/61`, the
  direct-root persistent Chromium regression passed `2/2`, and typecheck,
  artifact scan, A/B/C harness, and diff checks passed.
- `verified locally`: Web typecheck, the full `735/735` Vitest suite,
  demo/HTTP/production-offline builds, app-shell and build-artifact scans,
  web-server cache tests, A/B/C predecessor-bound artifact harness,
  demo-boundary smoke, and `git diff --check` passed. In-app Browser local
  preview was `blocked` by `ERR_BLOCKED_BY_CLIENT`; the repository-owned
  isolated Playwright lane supplied the service-worker evidence.
- `verified`: Surveil `9971f6b`, Workspace `500f025`, web image
  `sha256:02589efe…a3b40a`, gateway image `sha256:fd8c3bb1…694f`, and lock
  `sha256:0c587c48…9b1f05` were published. The allowlisted exact demo apply was
  `2 added, 5 changed, 2 destroyed`; public health and artifact hashes match
  the lock and the post-apply detailed plan returned literal `No changes`.
- `verified`: the hands-free successor uses Surveil `675fb9a`, web image
  `sha256:498a830a…3c804`, release fingerprint `sha256:ab671e7e…16502`,
  worker SHA-256 `sha256:7fd3b86e…ab7d9`, and release lock
  `sha256:5f03cd9d…16354`. The exact seven-action plan applied `2 added, 5
  changed, 2 destroyed`; instance `i-0babb12831e5a8beb` reached public ready
  HTTP 200, target-specific qualification smoke passed without business
  mutation, public artifact hashes matched the lock, and the post-apply
  detailed-exitcode plan returned literal `No changes`.
- `verified`: the retained Safari profile entered only through normal
  `https://demo.aviasurveil.com/` after the hands-free release. No recovery URL,
  hard refresh, or site-data clearing was used; the current sign-in shell
  rendered and the stale Planning shell did not return. The prior manually
  recovered transition means the exact active-plus-broken-waiting legacy state
  remains proven by the persistent Chromium regression rather than recreated
  destructively in Safari.
- `verified`: real Google Chrome used only normal root with no recovery URL,
  hard refresh, or site-data clearing. Its controller and active registration
  were `/sw.js` in `activated` state with no waiting/installing worker and
  `updateViaCache: none`; CacheStorage contained only the exact current
  `aviasurveil360-app-shell-ab671e7e…16502` cache. CDP showed the document,
  current bundles, CSS, and assets served from that worker/cache; meaningful
  sign-in DOM rendered with no overlay and zero console warnings/errors. The
  read-only sign-in action reached Avia Identity. Dashboard/Planning remains
  `not run` because no credentials were entered.
- `superseded historical blocker`: the earlier full Web run ended at
  `682 passed / 8 failed`; the fresh 2026-08-19 full run passed `735/735`.
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
6. Publish through the exact release flow and verify normal-root convergence
   in retained Safari and real Chrome. Keep
   `/app-shell-recovery.html?returnTo=%2F` as a diagnostic fallback, never the
   normal application URL.

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
failed candidate cleanup, network-only legacy recovery, quiescent client
reload, deferred predecessor-cache deletion, Chromium persistence, and WebKit
history behavior.

## Acceptance criteria

- No browser/worker path clears IndexedDB, OPFS, outbox, packages, or
  attachment manifests.
- Automatic activation requires exact complete-vector equality.
- Current-protocol clients remain waiting until the safe checkpoint is ACKed;
  after activation they perform a client-controlled automatic reload only
  while quiescent and are never force-navigated while local work may exist.
- A deployed client that cannot emit the current ACK converges through normal
  root entry even when a broken worker already waits: only an exact-vector
  successor activates, dirty tabs and durable storage remain intact, and
  quiescent tabs reload themselves without operator cache/site-data clearing.
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
change is a separate migration plan. The authorized demo rollout used only the
exact immutable lock and allowlisted Terraform plan; no broad Terraform action,
ad-hoc AWS infrastructure mutation, secret/DNS/tunnel change, preprod action,
or prod action was performed.

## Execution Prompt

Continue from this plan in the Surveil checkout. Preserve unrelated changes.
Re-run the focused gates, review only task-owned changes, then obtain explicit
authorization before commit, push, image publication, or the exact
`namibia/demo` release flow. After release, require `/sw.js?v=9` and `/sw.js` to
return the same current worker body before retained-profile Safari validation.
