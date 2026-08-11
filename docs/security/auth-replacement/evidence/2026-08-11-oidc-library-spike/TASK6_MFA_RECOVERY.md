# Task 6 MFA, recovery, verification, mail, and localization

Date: 2026-08-11

Status: `verified locally` for the bounded factor/challenge/domain components,
SMTP downgrade and injection rejection, invitation verification/resend, and
four-locale template rendering. The result remains `candidate-only`; no SMTP
provider or customer identity was contacted.

## Implemented boundary

- `internal/mfa` encrypts TOTP secrets with AES-GCM, supports RFC 6238 SHA-1
  codes with a bounded window, consumes counters monotonically, and returns
  the secret/URI only during enrollment. Recovery codes are random, hashed,
  single-use, and attempt-locked.
- `internal/challenge` issues random hashed tokens bound to subject and
  purpose (`email-verification`, `password-reset`, `mfa-recovery`, or
  `admin-recovery`) with expiry, single-use, and bounded rejection attempts.
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

Mailpit delivery/retry, real verified TLS SMTP, backup/restore of encrypted
factor material, end-user browser screens, and independent security review are
`not run`. No token, code, seed, SMTP credential, or customer data is present
in repository evidence.
