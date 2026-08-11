# Task 8 API and web adaptation

Date: 2026-08-11

Status: `verified locally` for the provider-neutral API/admin boundary, same-
origin BFF session contract, CSRF and safe return/logout handling, and browser
refresh serialization/terminal cleanup. The result remains `candidate-only`;
the normal Keycloak profile is preserved and no dual-issuer fallback is
enabled.

## Implemented boundary

- `apps/api/internal/identity/provider_admin.go` owns the neutral provisioning,
  directory, authority, lifecycle, required-action, MFA, and session-revocation
  method set. The Keycloak client is an adapter/baseline implementation and
  remains available for rollback; application code does not choose an issuer
  from a token or silently fall back between providers.
- The API's existing `OIDCProvider`, `AuthBoundary`, encrypted browser session,
  state/nonce/PKCE, exact provider logout ticket, CSRF, subject, membership,
  role, and organization predicates are unchanged. Browser JavaScript still
  receives only a same-origin session projection and never a provider refresh
  token.
- `apps/web/src/auth/RefreshCoordinator` shares one in-flight refresh promise,
  prevents a refresh stampede, clears local session state on terminal
  `REFRESH_REUSE`/`SESSION_REVOKED`/`AUTH_REVISION_STALE`/`UNAUTHENTICATED`, and
  permits retry after a non-terminal failure.

## Fresh verification

| Command | Result |
| --- | --- |
| `npm --prefix apps/web run typecheck` | `verified locally` |
| `npm --prefix apps/web test -- src/auth/session-client.test.ts` | `verified locally` — 13 tests passed, including CSRF, safe return/logout, same-origin projection, concurrent refresh dedupe, terminal cleanup, and retry behavior |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-api-go-cache go test ./internal/identity ./internal/administration -run 'TestProviderAdminBoundaryIsProviderNeutral\|^$' -count=1` | `verified locally` — neutral adapter boundary compiles and existing administration package remains buildable |

Full React route/accessibility/browser OIDC profile, long-lived channel
revocation, real replacement-provider API wiring, and both-profile end-to-end
qualification are `not run` here and remain Tasks 9–10/release gates. Keycloak
is still the explicit serving and rollback baseline.
