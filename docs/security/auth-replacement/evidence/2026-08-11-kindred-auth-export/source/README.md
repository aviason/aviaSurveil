# AviaSurveil360 authentication and identity boundary export

This is a sanitized, candidate-only source export of the authentication and identity boundary in `projectKindred`. It is intended for reuse and review, not deployment. It contains the cohesive server-side auth code, the auth-adjacent DynamoDB and consent code needed to understand it, selected tests, a source-reference copy of the mobile bearer client, contracts, and review reports.

## Source and scope

- Source revision: `cfcf14a6de6a5e7c00ff116dd47e477dddc68c74`
- Source branch: `main`
- Source tree at capture time: clean (`main...origin/main`, no tracked changes); the local `export_staging/` directory is the temporary untracked packaging artifact and is not part of the source revision.
- Candidate export generated: 2026-08-11 (Europe/Istanbul)
- Persistence boundary: DynamoDB; no PostgreSQL schema, migration, or PostgreSQL integration implementation exists in the source tree.

The export deliberately excludes `.git`, all `.env` files, credentials, tokens, private/signing keys, production or customer data, database dumps, deployment state, binaries, caches, dependency directories, seed/demo material, Postman environments, and raw infrastructure/mobile files containing operational URLs or API keys. `source-reference/dependency-map.md` records omitted transitive packages and why they are outside the auth boundary.

## Classification

The project is a proprietary RS256 JWT/session issuer with a minimal OpenID discovery/JWKS facade and a resource-server/consumer integration for API Gateway and AppSync. It is not a conforming OIDC Authorization Server/Provider and no local OIDC relying-party/client implementation was found. The mobile app is a custom email/password plus GraphQL bearer client. Publishing discovery or JWKS alone is not treated as provider compliance; see `reports/oidc-compatibility-report.md`.

## How to use this export

Start with `reports/capability-matrix.md`, `reports/architecture-and-data-model.md`, `reports/security-review.md`, and `reports/verification.md`. Go source is under `source/kindred_server`; tests are kept under `tests/kindred_server` to make their verification status explicit. The copied source is evidence-oriented and may require the omitted application packages listed in the dependency map to compile as a standalone module.

All findings and capability labels are candidate-only. No production-readiness claim is made.
