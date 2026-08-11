# Kindred Authentication Export Receipt

## Status

- Evidence status: `verified locally`
- Source status: `candidate-only`
- Runtime integration: `not run`
- OIDC provider conformance: `not established`
- Keycloak replacement: `not established`

## Source And Integrity

- Original temporary path:
  `/private/tmp/aviasurveil360-auth-oidc-export-2.zip`
- Retained archive: `aviasurveil360-auth-oidc-export-2.zip`
- SHA-256:
  `5de123c9bd8a711889e85b1876329540dee49423f64568c0a4bade4b5a4ff79b`
- Source revision:
  `cfcf14a6de6a5e7c00ff116dd47e477dddc68c74`
- Source status reported by the export: clean before temporary export staging
- Regular files retained: `112`; `manifest.json` covers `110` payload files
  and excludes the two manifest files as documented by the export.

The retained archive hash matches the owner-supplied digest. `unzip -t` passed.
The archive contained no duplicate, absolute, traversal, or symlink entry. Its
`manifest.sha256` verified every listed file. The retained extraction under
`source/` matched the independently inspected temporary extraction with
`diff -qr` and produced no differences. A fresh high-signal scan found no
private-key marker, AWS access-key pattern, or long assigned secret value.

## Classification

This source is a proprietary RS256 JWT and single-refresh-session issuer with
minimal discovery/JWKS consumer plumbing. It is neither a conforming OIDC
Provider/Authorization Server nor an OIDC relying party. Publishing discovery
and JWKS endpoints does not establish protocol conformance.

Useful reviewed inputs include Argon2id password handling, RS256 algorithm and
issuer/audience checks, token-version account revocation, hashed verification
and reset challenges, fixed-window distributed rate-limit mechanics, and
native-client refresh-call serialization and secure token storage concepts.
They remain reference input and require adaptation to AviaSurveil360's
PostgreSQL, BFF, application authorization, and provider-session contracts.

The following source paths must not be imported as working runtime behavior:

- the DynamoDB user-row persistence and unconditional full-row updates;
- the user-ID-prefixed refresh token and unauthenticated logout behavior;
- the non-atomic single-row refresh rotation;
- password change without all-session revocation;
- discovery metadata that advertises unsupported OIDC behavior;
- one-key signing without overlap rotation;
- fail-open or transport-incomplete auth throttling;
- session issuance that bypasses the configured email-verification gate;
- WebSocket authorization that is checked only at connect time; and
- legacy HMAC configuration beside the RS256 runtime.

## Fresh Intake Verification

| Check | Result |
|---|---|
| Supplied and retained archive SHA-256 | `verified locally`; exact expected digest |
| Archive compressed-data test | `verified locally`; no error |
| Duplicate, absolute, traversal, and symlink entry checks | `verified locally`; zero finding |
| Export `manifest.sha256` | `verified locally`; every listed file passed |
| Retained extraction versus inspected temporary extraction | `verified locally`; no difference |
| High-signal private-key, AWS-key, and assigned-secret scan | `verified locally`; zero matching file |
| Focused Go tests in this workspace | `blocked`; required module versions were unavailable to the offline task-owned cache and no network fetch was authorized |
| DynamoDB Local integration | `not run` |
| PostgreSQL integration | `not applicable` to this DynamoDB source; required for the Avia adaptation |
| OIDC protocol, fuzz, mobile, application integration, and ARM64 gates | `not run` |

The export's own verification report records passed Go unit and focused race
tests at the source revision. That upstream report is retained evidence; it is
not relabeled as a fresh AviaSurveil360 execution.

## Use Boundary

The archive and extraction are immutable design evidence outside every runtime
build. No source file is enabled by Compose or linked into an application. The
existing Auth Replacement ExecPlan controls any reviewed adaptation. Keycloak
remains required until protocol, security, PostgreSQL, migration, recovery,
rollback, and capacity gates pass and a separate cutover is authorized.
