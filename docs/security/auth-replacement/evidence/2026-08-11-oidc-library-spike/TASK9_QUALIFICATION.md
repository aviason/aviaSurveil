# Task 9 isolated local qualification and synthetic migration

Date: 2026-08-11

Status: `verified locally` for the isolated synthetic qualification harness,
opaque-subject manifest, membership-gated provider sessions, invitation
single-use, MFA/recovery, reset challenge binding, password-change revocation,
and suspension revocation. The result remains `candidate-only`; no real
identity, production profile, or Keycloak state was touched.

## Isolated profile

`apps/auth/internal/qualification/testdata/qualification-manifest.json` fixes
the candidate-only profile's distinct issuer, client, database name, cookie
prefix, HTTP/PostgreSQL/Mailpit ports, signing/MFA key IDs, and synthetic
opaque subjects. It contains only `.invalid` addresses and synthetic
organization labels. The existing opt-in Compose profile is
`deploy/local/auth/compose.auth-candidate.yaml`; the normal Keycloak profile
is unchanged.

## Qualification assertions

- Synthetic subjects are `usr_` identifiers validated independently of email,
  username, organization, or role and are unique in the manifest.
- Invitation verification is one-use; activation requires verified email and
  advances the auth revision.
- A valid provider identity with no Avia application membership cannot issue a
  provider refresh family. Adding membership enables issuance; membership is
  not stored as provider role authority.
- TOTP enrollment/replay, hashed recovery-code consumption, subject-bound
  reset challenge use/replay, password-change revision advance, family
  revocation, and suspension are exercised in one deterministic harness.

## Fresh verification

| Command | Result |
| --- | --- |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./internal/qualification -count=1 -v` | `verified locally` — manifest boundary and end-to-end synthetic qualification tests passed |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test -race ./internal/qualification -count=1` | `verified locally` |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./... -count=1` | `verified locally` — the qualification harness passes with every auth package |

Full Playwright browser login/logout, accessibility, restart/dependency-loss,
backup/restore, native ARM64 mixed-load, key-rotation rollback, and a complete
Keycloak rollback traffic exercise are `not run`. These remain required before
release and cannot be represented by this local deterministic harness.

## Fresh continuation boundary (2026-08-11)

The synthetic harness was rerun through the full auth package suite and race
suite and is `verified locally`. A full browser OIDC/MFA/recovery/reset/logout
qualification is `blocked`, not merely deferred: `apps/auth/internal/httpserver`
mounts only `/health/live`, `/health/ready`, and `/healthz`; it mounts no OIDC,
identity, MFA, recovery, reset, logout, or provider-admin route. The opt-in
candidate Compose file also contains only the auth service and is not wired to
the separate PostgreSQL/Mailpit topology. `scripts/test-auth-candidate-mailpit-outbox.sh`
has independently exercised the encrypted PostgreSQL outbox and TLS/STARTTLS
Mailpit retry path `verified locally`; it does not convert that component test
into an end-user runtime qualification.

`scripts/test-auth-candidate-postgres.sh` also applied `000005_mfa.up.sql` and
exercised the durable PostgreSQL MFA adapter `verified locally`: encrypted TOTP
storage, monotonic replay rejection, hashed single-use recovery codes, bounded
recovery failures, and reset. The adapter remains unmounted, so this does not
alter the end-user browser result, which is `blocked`.

The same isolated PostgreSQL suite applied `000006_challenges.up.sql` and
exercised durable challenge state `verified locally`: token hashes, exact
subject/purpose binding, expiry, attempt budgets, consume/invalidate/cleanup,
and concurrent single-use consumption. This adapter is also unmounted, so the
end-user browser result remains `blocked`.

The same isolated PostgreSQL suite applied `000007_oidc_runtime.up.sql` and
exercised the durable selected-library storage adapter `verified locally`:
exact-client authentication, encrypted signing-key retrieval, authorization
code state, and refresh rotation/reuse denial. No provider route mounts that
adapter, so end-user browser qualification remains `blocked`.

`internal/provider.RuntimeCandidate` now connects durable password and MFA
logic to the selected-library callback in process. The disposable PostgreSQL
suite is `verified locally` for a factor-free password login and for the
password → staged subject → TOTP → callback path. It is not mounted by
`cmd/auth`, therefore end-user browser qualification remains `blocked`.

No browser process, container, normal profile, Keycloak profile, real identity,
or external SMTP account was started or changed. The result remains
`candidate-only`; release is `release pending`.

## Isolated browser attempt (2026-08-12)

The task-owned browser profile completed discovery, authorization, login, and
MFA form submission in the disposable topology. The selected provider returned
the callback redirect, but the browser aborted the internal
`/authorize/callback` request with `net::ERR_ABORTED` before the local callback
was reached. Browser OIDC/MFA/recovery/reset/logout is `blocked`, not
`verified locally`. Cleanup removed the task-owned browser profile, processes,
containers, volumes, and secrets. This remains `candidate-only` and release is
`release pending`.

## Dependency-loss and restart attempt (2026-08-12)

The disposable candidate runner stopped task-owned authenticated STARTTLS
Mailpit and observed auth readiness return HTTP 503 while liveness remained
available. Mailpit restoration returned readiness to HTTP 200; a subsequent
auth-container restart also returned to readiness against its durable
PostgreSQL state. This is `verified locally` and `candidate-only`. Browser
OIDC/MFA/recovery/reset/logout remains `blocked`; release remains `release
pending`.

## Backup and restore attempt (2026-08-12)

`scripts/test-auth-candidate-backup-restore.sh` produced a task-owned database
dump, removed only the disposable privileged auth schema, restored it, and
restarted the auth runtime. Readiness returned and the synthetic active account
remained present. This is `verified locally`, `candidate-only`, and does not
change the browser OIDC/MFA/recovery/reset/logout `blocked` state or release
`release pending` status.

## Runtime topology continuation (2026-08-12)

`scripts/test-auth-candidate-runtime.sh` is `verified locally` for a dedicated
PostgreSQL, authenticated STARTTLS Mailpit, and auth candidate topology. It
observed healthy auth liveness, dependency-gated readiness, and OIDC discovery
from the topology's private network and removed all task-owned resources.
Browser OIDC/MFA/recovery/reset/logout remains `not run` because provider-owned
recovery, reset, and explicit logout handlers are not complete. This remains
`candidate-only`, release remains `release pending`, and Keycloak remains the
serving provider and rollback baseline.

The mounted `/recover/password` request was then exercised with one synthetic
`.invalid` account. Its generic response and encrypted-outbox delivery through
the runtime's authenticated STARTTLS Mailpit dependency are `verified locally`;
the one-time token was not emitted. The durable provider suite separately
consumed synthetic one-time challenges through the mounted password-reset and
MFA-reset handlers `verified locally`; browser flows remain `not run`, so this
remains `candidate-only` and release remains `release pending`.

## Browser qualification completion (2026-08-12)

`scripts/test-auth-candidate-browser.sh` is `verified locally` with a fresh
task-owned PostgreSQL/authenticated STARTTLS Mailpit Compose topology and an
ephemeral isolated Playwright context. It completed discovery, OIDC
Authorization Code with S256 PKCE and standard `form_post`, provider-owned
password login, TOTP MFA, token exchange, generic recovery initiation, password
reset, MFA reset, post-reset password login without MFA, and explicit logout.
Its cleanup removed task-owned browser, containers, volumes, and temporary
secrets. This is `candidate-only`; rollback traffic, native ARM64 mixed-load
capacity, fresh SBOM/license/vulnerability/provenance, and independent review
are `not run`. Keycloak remains serving and the rollback baseline; release
remains `release pending` and literal `NO-GO`.

## Native ARM64 bounded auth mixed-load qualification (2026-08-12)

`scripts/test-auth-candidate-load.sh` is `verified locally` on the `arm64`
host in the disposable candidate topology. It completed two concurrent
successful Authorization Code + S256 PKCE password logins at the configured
two-operation Argon2id capacity, two concurrent rejected unknown-account
Argon2id attempts, four recovery/outbox Mailpit receipts, and concurrent
readiness/discovery probes in 590 ms. The runner preserves the Argon2id capacity
bound and removes all task-owned state. This is `candidate-only`; it does not
cover the complete gateway/API/worker/PDF workload, so the release-level native
ARM64 capacity gate is `not run`. Keycloak remains serving and the rollback
baseline; release remains `release pending` and literal `NO-GO`.

## Candidate provider-form accessibility qualification (2026-08-12)

`scripts/test-auth-candidate-browser.sh` is `verified locally` for the
provider-owned login, MFA, recovery, password-reset, and MFA-reset forms. Its
isolated browser asserts English document language, one main landmark, one
level-one heading, one submit button, and accessible label/control pairs with
required non-checkbox inputs on every rendered form. This is `candidate-only`
provider-form semantic accessibility only; the complete product accessibility
matrix remains `not run`. Keycloak remains serving and the rollback baseline;
release remains `release pending` and literal `NO-GO`.

## Durable signing-key rotation qualification (2026-08-12)

`scripts/test-auth-candidate-postgres.sh` is `verified locally` for durable
candidate key rotation: encrypted new RSA material became active, the prior
public key remained available in finite JWKS overlap, and retirement removed
the elapsed overlap from the key set. This is `candidate-only`; key-custody
operations, Keycloak rollback traffic, and release evidence are `not run`.
Keycloak remains serving and the rollback baseline; release remains `release
pending` with literal `NO-GO`.
