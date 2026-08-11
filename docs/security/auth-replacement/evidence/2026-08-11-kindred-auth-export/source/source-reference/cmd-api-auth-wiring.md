# Sanitized API wiring reference

Source references (not copied verbatim): `kindred_server/cmd/api/main.go:148-162,302-304` and `cmd/api/jwt_bootstrap.go:19-56`.

The API process loads an RSA private key from configured environment/SSM/path sources (or generates a development key), derives a `kid`, constructs one signer/verifier and a one-key JWKS document, then registers custom auth routes plus health/JWKS/discovery and GraphQL adapters. No key ring, overlap window, retirement, rotation scheduler or old-key verification set is present. Operational key material and seed/bootstrap values are intentionally excluded.
