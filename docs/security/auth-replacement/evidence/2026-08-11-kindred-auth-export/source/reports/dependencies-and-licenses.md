# Dependencies and licenses

The server module is `github.com/kindred-app/kindred-server` and declares Go 1.25. Direct dependencies include AWS Lambda/SDK v2 services (DynamoDB, S3, Firehose, SNS, SSM), `github.com/golang-jwt/jwt/v5`, `github.com/google/uuid`, `github.com/vektah/gqlparser/v2`, `golang.org/x/crypto`, and the app-store validation package. The complete declared dependency graph is preserved in `source/kindred_server/go.mod` and `go.sum`.

The repository has no license files or machine-readable license inventory in the reviewed tree. License status is therefore not independently verified; consumers must perform their own dependency/license review before reuse. The AWS SSO OIDC SDK dependency observed transitively is unrelated to this service's OIDC provider behavior.
