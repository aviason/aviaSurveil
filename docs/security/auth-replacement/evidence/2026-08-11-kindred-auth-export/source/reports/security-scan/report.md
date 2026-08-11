# Security Review: projectKindred

## Scope

Offline standard security review of the authentication, identity, token, session, GraphQL, WebSocket and deployment boundary.

- Scan mode: repository
- Target kind: git_worktree
- Target ID: cfcf14a6de6a5e7c00ff116dd47e477dddc68c74
- Revision: cfcf14a6de6a5e7c00ff116dd47e477dddc68c74
- Snapshot digest: codex-security-snapshot/v1:sha256:0000000000000000000000000000000000000000000000000000000000000000
- Inventory strategy: repository
- Included paths: kindred_server/, kindred_mobile/kindred_swift/core/Services/, kindred_mobile/kindred_swift/core/Networking/, kindred_mobile/kindred_swift/core/Settings/
- Excluded paths: .git/, \*\*/.env\*, \*\*/infrastructure/\*/env.yaml, \*\*/seed.go, \*\*/build/, \*\*/.terraform/
- Runtime or test status: source-only; no application runtime or network access

Limitations and exclusions:
- No live deployment, PostgreSQL system, DynamoDB Local service, fuzz run, or external identity provider.
- Excluded \*\*/.env\*: Credentials, tokens and operational URLs excluded by user request.
- Excluded \*\*/infrastructure/\*/env.yaml: Deployment values and secret paths excluded.
- Excluded \*\*/seed.go: Demo/customer-like seed material excluded.
- Excluded PostgreSQL: No PostgreSQL implementation exists in the source tree.

### Scan Summary

| Field | Value |
| --- | --- |
| Reportable findings | 4 |
| Severity mix | high: 4 |
| Confidence mix | high: 4 |
| Coverage | partial |
| Validation mode | independent baseline plus focused source review |

Canonical artifacts: `scan-manifest.json`, `findings.json`, and `coverage.json`. This report is a deterministic projection of those files.

## Threat Model

No explicit canonical threat-model summary was recorded.

## Findings

| Finding | Severity | Confidence | Detailed write-up |
| --- | --- | --- | --- |
| [Refresh-token rotation is not conditional](#finding-1) | high | high | inline below |
| [Logout revokes a user selected by an unverified refresh-token prefix](#finding-2) | high | high | inline below |
| [Password change leaves existing sessions valid](#finding-3) | high | high | inline below |
| [Public GraphQL authentication mutations bypass the application limiter](#finding-4) | high | high | inline below |

### Confidence Scale

| Label | Meaning |
| --- | --- |
| high | Direct evidence supports the finding with no material unresolved blocker. |
| medium | Evidence supports a plausible issue, but material runtime or reachability proof remains. |
| low | Evidence is incomplete and the item is retained only for explicit follow-up. |

<a id="finding-1"></a>

### [1] Refresh-token rotation is not conditional

| Field | Value |
| --- | --- |
| Severity | high |
| Confidence | high |
| Confidence rationale | The repository update path lacks a prior-hash/version condition and no contention test exists. |
| Category | concurrency |
| CWE | CWE-362 |
| Affected lines | kindred_server/internal/auth/service.go:477-517, kindred_server/internal/platform/storage/dynamodb.go:187-201 |

#### Summary

A refresh request reads the current hash and later writes a replacement through an unconditional full-row update, so concurrent reuse is not proven single-use.

#### Validation

The repository update path lacks a prior-hash/version condition and no contention test exists. Validation details were not recorded separately.

#### Dataflow

The canonical finding records the affected path at kindred_server/internal/auth/service.go:477-517, kindred_server/internal/platform/storage/dynamodb.go:187-201, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**High** — The scan assigned high severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Rotate with a conditional update on the prior hash/token version and test concurrent refresh/reuse.

<a id="finding-2"></a>

### [2] Logout revokes a user selected by an unverified refresh-token prefix

| Field | Value |
| --- | --- |
| Severity | high |
| Confidence | high |
| Confidence rationale | Direct source trace from the logout input to token-version and refresh-state writes. |
| Category | broken-access-control |
| CWE | CWE-639 |
| Affected lines | kindred_server/internal/auth/service.go:520-546, kindred_server/internal/auth/handler.go:32-39 |

#### Summary

Logout parses a user identifier from caller-controlled refresh-token text without verifying token ownership, enabling cross-account session invalidation.

#### Validation

Direct source trace from the logout input to token-version and refresh-state writes. Validation details were not recorded separately.

#### Dataflow

The canonical finding records the affected path at kindred_server/internal/auth/service.go:520-546, kindred_server/internal/auth/handler.go:32-39, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**High** — The scan assigned high severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Bind logout to authenticated subject and verify the complete refresh token hash before revocation.

<a id="finding-3"></a>

### [3] Password change leaves existing sessions valid

| Field | Value |
| --- | --- |
| Severity | high |
| Confidence | high |
| Confidence rationale | Password reset closes sessions in a sibling path, while ChangePassword omits that operation. |
| Category | session-management |
| CWE | CWE-613 |
| Affected lines | kindred_server/internal/auth/service.go:701-723 |

#### Summary

The authenticated password-change path updates only the password hash and does not invalidate token version or refresh state.

#### Validation

Password reset closes sessions in a sibling path, while ChangePassword omits that operation. Validation details were not recorded separately.

#### Dataflow

The canonical finding records the affected path at kindred_server/internal/auth/service.go:701-723, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**High** — The scan assigned high severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Close sessions and bump token version atomically on every password change; test old access and refresh tokens.

<a id="finding-4"></a>

### [4] Public GraphQL authentication mutations bypass the application limiter

| Field | Value |
| --- | --- |
| Severity | high |
| Confidence | high |
| Confidence rationale | Runtime wiring and schema/resolver paths show separate GraphQL execution without the limiter middleware. |
| Category | missing-rate-limit |
| CWE | CWE-307 |
| Affected lines | kindred_server/internal/graphql/auth_resolvers.go:106-171, kindred_server/cmd/api/main.go:166-178 |

#### Summary

The configured auth limiter is wired to legacy REST routes, while AppSync public login, registration, refresh and reset mutations call the service directly.

#### Validation

Runtime wiring and schema/resolver paths show separate GraphQL execution without the limiter middleware. Validation details were not recorded separately.

#### Dataflow

The canonical finding records the affected path at kindred_server/internal/graphql/auth_resolvers.go:106-171, kindred_server/cmd/api/main.go:166-178, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**High** — The scan assigned high severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Apply distributed per-IP/account/device throttles to every public GraphQL auth mutation.

## Reviewed Surfaces

| Surface | Risk Area | Outcome | Notes |
| --- | --- | --- | --- |
| Credential, JWT and refresh-session flows | authentication | Reported | No additional canonical notes were recorded. |
| GraphQL/API-key/OIDC identity boundary | authorization | Reported | No additional canonical notes were recorded. |
| WebSocket bearer sessions | session management | Reported | No additional canonical notes were recorded. |
| JWT/JWKS/CORS/IAM deployment wiring | configuration | Reported | No additional canonical notes were recorded. |

## Open Questions And Follow Up

- Is the single refresh session per user an intentional product invariant, and is the email-verification gate intended to cover registration-issued sessions?
- No live deployment or external identity provider was accessed.
  - Follow-up prompt: Review deferred unit deferred_live_runtime and close its stated proof gap.
- DynamoDB Local contention and crash-recovery tests were not run.
  - Follow-up prompt: Review deferred unit deferred_dynamodb_contention and close its stated proof gap.
