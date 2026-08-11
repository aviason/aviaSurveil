# Task 6 MFA, recovery, verification, mail, and localization

Date: 2026-08-11

Status: `verified locally` for bounded factor/challenge/domain components,
durable encrypted PostgreSQL MFA state, SMTP downgrade and injection rejection,
invitation verification/resend, and four-locale template rendering. The result
remains `candidate-only`; no SMTP provider or customer identity was contacted.

## Implemented boundary

- `internal/mfa` encrypts TOTP secrets with AES-GCM, supports RFC 6238 SHA-1
  codes with a bounded window, consumes counters monotonically, and returns
  the secret/URI only during enrollment. Recovery codes are random, hashed,
  single-use, and attempt-locked.
- `internal/mfa.PostgresStore` persists AES-GCM ciphertext, enabled state, a
  monotonic TOTP counter, recovery-failure state, and only SHA-256 recovery-code
  hashes in the privileged `auth_identity` schema. Its mutations lock the factor
  row; reset deletes the factor and its cascading recovery-code state.
- `internal/challenge` issues random hashed tokens bound to subject and
  purpose (`email-verification`, `password-reset`, `mfa-recovery`, or
  `admin-recovery`) with expiry, single-use, and bounded rejection attempts.
- `internal/challenge.PostgresStore` persists only token hashes and applies
  exact subject/purpose binding, expiry, attempt budgets, consume,
  invalidation, cleanup, and row-locked transitions in `auth_identity`.
- Invitation verification consumes one token, marks email verified, advances
  the auth revision, and never returns the raw token from a later snapshot.
  Resend invalidates the predecessor and enforces a bounded count.
- `internal/mail` accepts only TLS/STARTTLS SMTP with TLS 1.2+, strict
  recipient/from/header/body validation, context timeouts, and no plaintext
  downgrade. `internal/localization` has safe `en`, `tr`, `fr`, and neutral
  Portuguese catalogs with required-key/template checks.

## Fresh verification

| Command | Result |
| --- | --- |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./internal/mfa ./internal/challenge ./internal/mail ./internal/localization ./internal/identity -count=1` | `verified locally` — enrollment, encrypted-at-rest boundary, replay, recovery consumption/lock, challenge binding/expiry, invitation single-use/resend, SMTP safety, and all locale templates passed |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test -race ./internal/mfa ./internal/challenge ./internal/mail ./internal/localization ./internal/identity -count=1` | `verified locally` |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go vet ./...` | `verified locally` |
| `scripts/test-auth-candidate-postgres.sh` | `verified locally` — disposable PostgreSQL applied `000005_mfa.up.sql` and passed encrypted TOTP persistence, replay rejection, hashed single-use recovery codes, bounded recovery failures, and reset |
| `scripts/test-auth-candidate-postgres.sh` challenge extension | `verified locally` — disposable PostgreSQL applied `000006_challenges.up.sql` and passed hashed storage, subject/purpose binding, expiry, attempt locking, invalidation, cleanup, and concurrent single-use consumption |
| `scripts/test-auth-candidate-postgres.sh` provider MFA extension | `verified locally` — an enabled factor stages the password-authenticated subject and requires durable TOTP completion before the OIDC callback; recovery-code selection uses the same bounded durable factor state |

The encrypted PostgreSQL outbox and real verified TLS/STARTTLS Mailpit
delivery/retry are `verified locally`: a transient failure becomes a bounded
retryable lease, the retry delivers to Mailpit, and its receipt API confirms
delivery. The outbox stores recipient, subject, and body encrypted at rest;
raw verification/reset values are not recorded in evidence. Backup/restore of
encrypted factor material, end-user browser screens, and independent security
review are `not run`. MFA remains `blocked` from end-user qualification because
the auth server still mounts only health routes. No token, code, seed, SMTP
credential, or customer data is present in repository evidence.

## Runtime continuation (2026-08-12)

The isolated candidate runtime now mounts the durable password-login and MFA
handlers after PostgreSQL initialization; topology readiness is `verified
locally` in a disposable PostgreSQL/Mailpit run. End-user recovery and password
reset handlers are `not run`, and no recovery email was issued by this runtime
check. The result remains `candidate-only`, Keycloak remains the serving and
rollback baseline, and release remains `release pending`.

`scripts/test-auth-candidate-runtime.sh` now also posts a generic password
recovery request for one disposable `.invalid` active account and observes the
runtime outbox worker's authenticated, certificate-verified STARTTLS Mailpit
receipt `verified locally`. The raw one-time token is not printed. The
The token-consumption password-reset and MFA-reset handlers are `verified
locally` in the durable provider PostgreSQL suite: the reset password
authenticates and the MFA factor is removed. Browser qualification remains
`not run`. The result is `candidate-only` and release is `release pending`.

## Browser recovery/reset qualification (2026-08-12)

`scripts/test-auth-candidate-browser.sh` is `verified locally` in the
disposable candidate topology. It completed generic recovery initiation,
password-reset token consumption, MFA-reset token consumption, and a subsequent
password login that proceeded to the OIDC callback without MFA. No raw recovery
token, reset password, MFA seed, SMTP credential, customer identity, or normal
profile was used or recorded. This is `candidate-only`; Keycloak remains the
serving and rollback baseline, and release remains `release pending`.
