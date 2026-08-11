# Omitted transitive dependency map

The export includes the auth boundary and its security-relevant adjacent types, but excludes unrelated business-domain packages and operational material. These references prevent silent omission of dependencies:

| Included boundary file | Omitted dependency | Why it is omitted / what to inspect |
|---|---|---|
| `internal/platform/storage/dynamodb.go` | request/points storage types (`request_dynamodb.go`, `points` models, pagination helpers) | Product-domain records share the DynamoDB repository; they are not identity state. The copied file shows the user/lock/session operations. Minimal type/source-reference copies are under `source-reference/kindred_server/internal/{request,points,platform/ddbpagination}`. |
| `internal/graphql/auth_resolvers.go` | product points/stats/subscription resolvers and generated GraphQL glue | Auth mutations are interleaved with product queries; minimal interface/model source-reference copies are under `source-reference/kindred_server/internal/{points,stats,subscription}`. |
| `internal/messagews/*` | request/message/messagefanout implementations | WebSocket auth/session behavior is copied; minimal request/message/fanout model/source-reference files are under `source-reference/kindred_server/internal/{request,message,messagefanout}`. |
| copied auth tests under `tests/` | repository-wide test fakes and fixtures | The selected tests are copied, but shared fakes depend on unrelated application repositories and seed data; no credentials or demo material is exported. |
| `cmd/api` wiring | `main.go`, seed/bootstrap and deployment env files | Raw wiring includes operational URLs, secret paths and seed data. `source-reference/infrastructure/auth-terraform-extract.md` and `source-reference/cmd-api-auth-wiring.md` preserve the auth-relevant control flow. |
| mobile app container | `AppContainer.swift` | It embeds production/preproduction URLs and API-key-like values. Sanitized client files and an OIDC-boundary note are exported instead. |

These omissions are deliberate, documented, and not evidence that the omitted business capabilities are part of authentication.
