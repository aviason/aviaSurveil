# Authentication Replacement Hardening Context

## Evidence inventory

| ID | Evidence | Integrity and relevance |
|---|---|---|
| `AUTH-EXPORT-1` | First-party PostgreSQL-oriented authentication export | Exact archive SHA-256 `7fa982300440cb3e79d28bc0f7f22ebb59124bc9c125dededb22dea306fc7fb7`; 30 payload files; source revision `60dbe494318106569f6d9dbea121d6b1c841ae95`. |
| `AUTH-REVIEW-1` | First export security review and capability matrix | Records 18 validated findings, missing OIDC/MFA/recovery/tenant controls, and blocked PostgreSQL verification. |
| `AUTH-EXPORT-2` | Kindred RS256 JWT/session export | Exact archive SHA-256 `5de123c9bd8a711889e85b1876329540dee49423f64568c0a4bade4b5a4ff79b`; 112 retained regular files; source revision `cfcf14a6de6a5e7c00ff116dd47e477dddc68c74`. |
| `AUTH-REVIEW-2` | Kindred security, OIDC, refresh, mobile, and architecture reports | Confirms that discovery/JWKS is not OIDC conformance and identifies refresh atomicity, logout authorization, password-change revocation, stale WebSocket, key rotation, verification, throttling, and configuration-drift gaps. |
| `AVIA-IDENTITY` | Current AviaSurveil360 identity/session implementation | Current dirty working tree at Git HEAD `e36a395492f82b2bd4c79d2b580449fddbedf07e`; exact working-tree drift is present and must be refreshed before implementation. |
| `AVIA-POLICY` | Current identity/data profile and MFA runbook | Establishes public-registration denial, one exact organization and role, provider observation, TOTP, subject retention, fail-closed sessions, and recovery boundaries. |

The combined evidence collection SHA-256 is
`06e71b74208ee3e703ecd096a3ba126b6dd3b4e1509850e8a06be38098a8b175`.
It is the SHA-256 of the two exact archive labels and digests, sorted bytewise
from these canonical lines; it is not a replacement for either archive digest:

```text
first-party-auth-export 7fa982300440cb3e79d28bc0f7f22ebb59124bc9c125dededb22dea306fc7fb7
kindred-auth-export 5de123c9bd8a711889e85b1876329540dee49423f64568c0a4bade4b5a4ff79b
```

## Inspected source boundaries

- First source evidence:
  `../evidence/2026-08-11-first-party-auth-export/source/auth-export/`
- Second source evidence:
  `../evidence/2026-08-11-kindred-auth-export/source/`
- Comparative adaptation decision: `source-comparison.md`
- Current OIDC client: `apps/api/internal/identity/oidc_remote.go`
- Current provider administration: `apps/api/internal/identity/keycloak_admin.go`
- Current browser-session boundary: `apps/api/internal/platform/session/`
- Current HTTP/BFF boundary: `apps/api/internal/httpapi/auth.go`
- Current React session client: `apps/web/src/auth/session-client.ts`
- Current provider configuration: `deploy/local/keycloak/` and
  `deploy/aws-private-pilot/compose.yaml`

Neither imported source is a conforming OIDC provider. The first contributes
PostgreSQL row-lock and refresh-family concepts but duplicates application
authorization. The second contributes RS256/JWKS negative-test ideas, account
lifecycle cases, and native-client refresh serialization, but stores all
security state in a DynamoDB user row with unsafe full-row concurrency and no
tenant model. AviaSurveil360 already owns application sessions, CSRF, desired
membership, exact role/organization validation, immutable identity references,
and provider reconciliation. The replacement preserves those controls rather
than importing either source's persistence or authority model.

## Evidence limitations

- The imported PostgreSQL migration and store concurrency paths have not been
  exercised against a task-owned PostgreSQL database.
- The Kindred source's focused Go tests were `blocked` in this workspace because
  the exact module versions were unavailable to the offline task-owned cache.
  Its retained source report records upstream unit and race results, while
  DynamoDB Local, fuzz, mobile, and OIDC protocol tests remain `not run` or
  absent.
- The current AviaSurveil360 working tree contains extensive user and main-task
  changes, so implementation source drift is `present`.
- No current online vulnerability scan, OIDC conformance certification,
  penetration test, ARM64 capacity measurement, or production migration has
  been run for the proposed design.
- The owner's first-party code ownership decision removes the source-license
  blocker; third-party dependency notices and current security advisories still
  require verification.
