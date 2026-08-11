# Authentication Replacement Security Package

This package keeps the first-party source evidence separate from derived design
and implementation planning.

- [`evidence/2026-08-11-first-party-auth-export/IMPORT_RECEIPT.md`](evidence/2026-08-11-first-party-auth-export/IMPORT_RECEIPT.md)
  records ownership, integrity, and the first candidate-only use boundary.
- [`evidence/2026-08-11-kindred-auth-export/IMPORT_RECEIPT.md`](evidence/2026-08-11-kindred-auth-export/IMPORT_RECEIPT.md)
  records integrity, classification, and the use boundary for the second
  owner-provided auth/JWT/session export.
- [`hardening/source-comparison.md`](hardening/source-comparison.md) selects the
  reusable concepts and adds source-derived regression contracts without
  importing either candidate runtime.
- [`hardening/hardening.md`](hardening/hardening.md) presents the reviewed
  architecture options and recommendation.
- [`hardening/proposals/first-party-identity-boundary.md`](hardening/proposals/first-party-identity-boundary.md)
  contains the complete security and engineering tradeoff analysis.
- [`hardening/implementation/separate-go-oidc-provider.md`](hardening/implementation/separate-go-oidc-provider.md)
  hands the selected design to the active ExecPlan.
- [`../../exec-plans/active/2026-08-11-first-party-go-oidc-auth-replacement-plan.md`](../../exec-plans/active/2026-08-11-first-party-go-oidc-auth-replacement-plan.md)
  controls implementation and verification.

The retained exports are not part of any runtime build. Keycloak remains the
active identity provider until the ExecPlan's cutover gates and separate exact
authorizations are complete.
