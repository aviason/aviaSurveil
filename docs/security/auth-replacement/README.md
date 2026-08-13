# Authentication Replacement

The repository now consumes one identity implementation: the shared first-party
Go OIDC service in `../../../../shared/auth`.

Current authorities:

- [First-Party Go OIDC Authentication And Repository-Local Provider Retirement](../../exec-plans/active/2026-08-11-first-party-go-oidc-auth-replacement-plan.md)
  controls implementation, retirement, and final verification.
- [Task 11 cutover and local-retirement evidence](evidence/2026-08-11-oidc-library-spike/TASK11_LOCAL_PREPROD_CUTOVER.md)
  records the canonical disposable topology and the later owner-authorized
  repository-local retirement.

Obsolete source exports, comparison spikes, hardening alternatives, rollback
baselines, and pre-cutover evidence were removed by explicit owner direction.
The maintained result is `candidate-only` and `release pending`. No remote,
deployment, traffic, production-secret, or real-user action is authorized by
this package.
