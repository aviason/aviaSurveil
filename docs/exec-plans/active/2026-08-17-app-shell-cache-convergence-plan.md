# App-Shell Cache And Exact-Vector Convergence

Date: 2026-08-17
Last updated: 2026-08-17
Status: active — implementation started; `candidate-only`; `release pending`

## Planning authority

This plan is governed by [`docs/PLANS.md`](../../PLANS.md). The root
cross-repository plan owns Workspace composition and release boundaries. The
Auth child plan owns provider transaction and authorization-code security.

## Objective and user-visible outcome

Generate a content-bound app-shell fingerprint and complete file manifest,
activate only exact-vector-compatible Service Workers, preserve durable offline
data, keep gateway-owned routes network-only, and remove the raw stale-login
failure without reviving any OIDC artifact.

The deployed v9 legacy bridge deliberately does not force-reload an
unacknowledged legacy document. Fingerprint-aware clients reload only after
their dirty/in-flight/quiescence acknowledgement.

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
- `blocked`: the full web suite ended at `674 passed / 8 failed`; failures are in unrelated dirty planning/management presentation surfaces, not the focused app-shell tests.
- `verified locally`: the Auth child migration/code-claim gate passed with no skipped required PostgreSQL tests.
- `blocked`: the in-app Browser rejected the local Caddy internal CA; clean HTTP preview browser coverage passed for root, lazy Inspector route, and mobile route.
- `not run`: OCI inspection/public release lock publication, Cloudflare discovery/purge, public transition, and demo apply.

## Ordered implementation

1. Finish the dependency-free vector, strict manifest/fingerprint generation,
   candidate CacheStorage validation, positive route policy, and quiescence
   protocol.
2. Add Auth code claim/finalization before stale-login recovery. Do not ship
   stale recovery if the concurrent one-shot test is unavailable or fails.
3. Add web-server/Caddy cache headers and explicit `/http-config.json`
   no-store fetching.
4. Add isolated A/B/C browser tests using task-unique project/state/ports.
5. Run native Surveil and Auth source gates; report missing fixtures literally.

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
```

The harness must prove A/B/C manifest determinism, per-file hash validation,
failed candidate cleanup, retained lazy chunks, legacy-v9 no-forced-reload,
quiescence-gated reload, Chromium persistence, and WebKit history behavior.

## Acceptance criteria

- No browser/worker path clears IndexedDB, OPFS, outbox, packages, or
  attachment manifests.
- Automatic activation requires exact complete-vector equality.
- Legacy v9 is never force-navigated without a client acknowledgement.
- `/api`, `/v1`, `/auth`, `/identity`, `/health`, `/operations`, `/otel`,
  private routes, and `/http-config.json` remain network-only.
- Every committed app-shell response is validated for origin, redirect status,
  media type, byte size, and SHA-256.
- All required gates use existing commands or commands added by this plan;
  `scripts/test-http-profile.sh` is not referenced.

## Recovery and release boundary

Failed local candidate installation deletes only its candidate cache. A vector
change is a separate migration plan. Public rollout, image publication,
Cloudflare purge, Terraform apply, and mutating qualification require separate
authorization and remain `release pending`.

## Execution Prompt

Continue from this plan in the Surveil checkout. Preserve unrelated changes and
do not change cloud locks or external state. Complete the browser/source slices,
run the exact local gates, and stop with literal `not run` or `blocked` labels
when Auth fixtures or safe external evidence are unavailable.
