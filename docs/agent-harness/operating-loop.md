# AviaSurveil360 Operating Loop

1. Read `AGENTS.md`, the active plan, product source, and this harness index.
2. Change the smallest owning surface; keep the legacy demo and candidate
   runtime boundaries distinct.
3. Select the smallest applicable local command from the verification matrix.
4. Run `make harness-maintenance` whenever harness authority or evidence routes change.
5. Record durable gaps in the active plan/index or technical-debt tracker, not
   only in chat.
6. Stop before Git writes, deployments, external fixtures, Data harness work,
   source/attestation commits, or HMAC-key handling unless explicit current
   authority names that action.

Use `verified locally` only for fresh local output. Use `not run` for an
unavailable fixture or higher-scope gate, `candidate-only` for local candidate
runtime work, and `blocked` for absent owner, commit, attestation, or
caller-owned external HMAC key authority. Recover from local failures with the
smallest documented retry/down command; do not broaden the task to production
or external systems.
