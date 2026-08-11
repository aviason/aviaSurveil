# Sanitized infrastructure/auth reference

This is a source-reference summary, not a deployment file. Original evidence: `kindred_server/infrastructure/modules/kindred-server/main.tf:118-124,1463-1508,1520-1530,1555-1578`, `variables.tf`, `outputs.tf`, and module tests.

- Public platform routes expose health, `/.well-known/jwks.json`, and `/.well-known/openid-configuration`.
- API Gateway can use a JWT authorizer configured with the service issuer, audience, and JWKS URI. The authorizer is conditional on a configured signing-key SSM path; the disabled path can leave edge routes unauthenticated while Lambda still performs its own checks.
- AppSync is configured as an `OPENID_CONNECT` consumer of the service issuer. This is consumer/resource-server integration, not an OIDC provider implementation.
- CORS configuration permits wildcard origins in the reviewed Terraform source; no browser cookie/BFF/CSRF boundary is implemented.
- Terraform still carries legacy `TOKEN_SECRET`/`token_secret_ssm_path` plumbing even though the current Go signer is RS256 and does not read that setting.

All operational URLs, secret paths, environment values and customer/deployment data are omitted.
