# App-Shell Cache And Exact-Vector Convergence

Date: 2026-08-17
Last updated: 2026-08-17
Status: active — legacy recovery and continuous-update implementation `verified locally`; `candidate-only`; `release pending`

## Planning authority

This plan is governed by [`docs/PLANS.md`](../../PLANS.md). The root
cross-repository plan owns Workspace composition and release boundaries. The
Auth child plan owns provider transaction and authorization-code security.

## Objective and user-visible outcome

Generate a content-bound app-shell fingerprint and complete file manifest,
activate only exact-vector-compatible Service Workers, preserve durable offline
data, keep gateway-owned routes network-only, and retire the legacy v9 worker
and document at cutover without reviving any OIDC artifact.

The strict retirement bridge force-navigates same-origin legacy clients after
the exact successor activates. It retains durable IndexedDB/OPFS/outbox state,
but does not promise preservation of unsaved in-memory legacy form state.

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
- `verified live diagnosis`: `https://demo.aviasurveil.com/` serves the current
  HTML while `/sw.js?v=9` still returns `410`; the public environment therefore
  does not contain this repair yet.
- `blocked`: the fresh full web suite ended at `682 passed / 8 failed`;
  failures are in unrelated dirty planning/management presentation surfaces,
  not the focused app-shell tests.
- `verified locally`: the Auth child migration/code-claim gate passed with no skipped required PostgreSQL tests.
- `blocked`: the in-app Browser rejected the local Caddy internal CA; clean HTTP preview browser coverage passed for root, lazy Inspector route, and mobile route.
- `not run`: Playwright WebKit because the local WebKit binary is unavailable;
  the same persistent-browser test supports
  `AVIA_LEGACY_UPDATE_BROWSER=webkit` when that binary is present.
- `not run`: commit, push, OCI inspection/public release lock publication,
  Cloudflare discovery/purge, public transition, and demo apply.

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
6. Publish one immutable release containing both the gateway legacy-URL bridge
   and the successor worker, then verify `/sw.js?v=9` returns the exact current
   worker body before testing a retained Safari profile.

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
- Legacy v9 clients are force-navigated after exact successor activation;
  unacknowledged legacy pages are not allowed to remain active.
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
