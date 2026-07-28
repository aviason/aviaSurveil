# Local Identity And MFA Recovery

This procedure covers local Keycloak, OIDC, session, TOTP, and exact role-scope
evidence for the `candidate-only` stack. It is not production-ready and is
`not run` for a new incident until browser verification succeeds.

## Scope And Owner

Owner: Platform/Operations

Escalation owner: Security and Release authority

Identity owns issuer, realm, MFA, and subject identity. Backend owns session
exchange and application role enforcement.

## Preconditions

- Confirm the exact project, state directory, HTTPS port, realm, client, and
  affected local subject.
- Keep TOTP seeds and bootstrap credentials outside logs and evidence.
- Distinguish authentication failure from application authorization failure.

## Symptoms

- OIDC discovery, callback, session exchange, or logout fails.
- TOTP enrollment or a normal TOTP login fails.
- The authenticated subject has missing, excess, or wrong organization scope.

## Safety Boundary

- Never disable MFA or broaden a role to make a check pass.
- Never replace one subject identity with another or reuse a TOTP seed.
- Do not expose bootstrap credentials, client secrets, cookies, or TOTP values.

## Diagnosis

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-identity-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
export AVIA_LOCAL_HTTPS_PORT="18443"
curl --fail --silent --show-error --insecure "https://localhost:$AVIA_LOCAL_HTTPS_PORT/identity/realms/aviasurveil360/.well-known/openid-configuration"
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full logs --no-color keycloak api
```

## Expected Output

Discovery names the configured local issuer. Logs show no credential values.
The browser flow retains exact subject, role, organization, and route
authorization; a denied role remains denied.

## Reversible Mitigation

Restart only the identity service and wait for its healthcheck:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-identity-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
export AVIA_LOCAL_HTTPS_PORT="18443"
export AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:$AVIA_LOCAL_HTTPS_PORT"
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full restart keycloak
docker compose --project-name "$AVIA_LOCAL_PROJECT" --file deploy/local/compose.yaml --profile full up --detach --wait keycloak api
```

If realm data or role scope differs, stop and escalate rather than editing it.

## Recovery Verification

Run the isolated OIDC/TOTP contract:

```bash
./scripts/test-http-oidc-profile.sh
```

Require normal TOTP login, exact role scope, negative authorization checks, no
secret leak, and zero task-owned residue before recording `verified locally`.

## Evidence Capture

Capture issuer, realm, client ID, opaque subject ID, expected/actual role and
organization scope, UTC timeline, negative authorization result, and cleanup
status. Exclude credentials, cookies, tokens, and TOTP material.

## Escalation

Escalate subject, role, organization, MFA, or realm mismatch to Security and
Identity. Escalate application policy mismatch to Backend and Product/CAA
Operations.

## Authorization Required

MFA reset, role or organization reassignment, user lifecycle changes, realm
import, credential rotation, production federation, and production account
operations require new explicit authorization.
