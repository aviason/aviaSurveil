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
