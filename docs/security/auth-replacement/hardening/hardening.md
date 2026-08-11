# Security Hardening Review: First-Party Authentication Replacement

## Evidence Basis

I inspected both integrity-retained owner-provided exports, their security,
capability, OIDC, refresh, architecture, and verification reports, and the
current AviaSurveil360 OIDC, Keycloak administration, browser-session, CSRF,
organization, and role boundaries. The archives are bound to SHA-256
`7fa982300440cb3e79d28bc0f7f22ebb59124bc9c125dededb22dea306fc7fb7`
and
`5de123c9bd8a711889e85b1876329540dee49423f64568c0a4bade4b5a4ff79b`.
The current application tree is dirty relative to Git HEAD, so implementation
must refresh source drift before it modifies identity code.

The exports are valuable source input: together they provide Argon2id, random
subject/session identifiers, RS256 negative checks, refresh row-lock/family
concepts, account lifecycle cases, and client-side refresh serialization. They
are not conforming OIDC providers. One duplicates application users and staff
permissions; the other combines security state in a DynamoDB user row with
unsafe full-row concurrency and no organization model. Neither persistence nor
authority model matches AviaSurveil360.

## Constraints

We must preserve immutable subject identity, exactly one valid organization and
role, server-owned application authorization, the same-origin BFF and CSRF
boundary, immediate lifecycle revocation, TOTP and recovery, four-language
identity messages, redacted durable auditing, and fail-closed provider
reconciliation. We must not create an issuer fallback or build cryptographic
and OIDC protocol mechanics from scratch. Keycloak remains the serving provider
until replacement parity and rollback are proven.

## Opportunity Portfolio

| Opportunity | Evidence | Options | Recommendation | Proposal |
|---|---|---|---|---|
| Replace Keycloak without collapsing authentication into application authorization | Both imported auth security reviews, the [source comparison](source-comparison.md), current OIDC/BFF/session source, and identity profile contract | 1. retain Keycloak; 2. embed auth in API; 3. separate Go OIDC provider | Option 3, with Keycloak retained through acceptance | [First-party identity boundary](proposals/first-party-identity-boundary.md) |

## Recommendation Summary

I recommend a small, separately privileged Go OIDC provider. This preserves the
existing protocol and BFF shape and keeps password verification, MFA secrets,
and signing authority outside ordinary API and worker code. It can reuse
selected exported primitives after their tactical defects are fixed, but it
must not import the first export's evaluation migration, the second export's
DynamoDB repository, or either proprietary JWT metadata surface as OIDC.

Embedding everything in the API is attractive for minimum memory and container
count, but the resulting credential and signing blast radius is disproportionate
for a surveillance product. Retaining Keycloak remains the safest fallback and
becomes the preferred outcome if the Go provider cannot pass protocol,
recovery, independent security, or ARM64 headroom gates.

## Next Decisions

- Execute the [selected implementation handoff](implementation/separate-go-oidc-provider.md)
  through the repository ExecPlan.
- Select a maintained OIDC authorization-server library only after a bounded
  interoperability, maintenance, and license spike.
- Freeze and implement the comparative `AS360-AUTH-001` through
  `AS360-AUTH-030` regression map before adapting source code.
- Confirm whether first cutover requires WebAuthn or whether TOTP plus hashed
  recovery codes is sufficient.
- Determine whether any non-synthetic Keycloak identities will exist at
  cutover; if so, design their migration as a separate approval-bound operation.
