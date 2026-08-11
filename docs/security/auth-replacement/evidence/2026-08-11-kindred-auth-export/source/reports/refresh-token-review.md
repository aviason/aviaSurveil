# Refresh-token behavior review

This is the requested refresh checklist, based on source evidence and the tests that completed in the original revision. “Verified locally” means the repository unit/full/race runs passed; it does not imply a concurrency or production deployment proof.

| Check | Classification | Evidence / test gap |
|---|---|---|
| Entropy and transport | implemented and verified locally | `crypto/rand` produces a 32-byte opaque secret; the token is returned through the JSON auth contract and is expected to travel over the deployment transport. Auth/full tests passed; HTTPS enforcement is infrastructure-dependent. |
| Hash/encryption at rest | implemented and verified locally | Only SHA-256 refresh hash is persisted; plaintext is not stored. No application-level encryption is implemented; storage encryption is deployment responsibility. |
| Family and session binding | partial | User row stores one hash, idle/absolute expiry and device; there is no token-family table or multi-session graph. |
| Client/device binding | partial | Optional client-supplied device ID is compared and mismatch clears the stored session; no durable client registry or strong device attestation. Device-mismatch tests passed. |
| Rotation atomicity | partial | Read/compare/write is implemented, but the DynamoDB update is an unconditional full-row write rather than a conditional prior-hash/version compare-and-swap. No concurrency test proves single-use behavior. |
| Concurrent refresh | blocked | No dedicated concurrent refresh test; race tests cover selected packages but not a DynamoDB contention scenario. |
| Reuse detection | partial | Hash/device mismatch clears the one stored session; explicit old-token replay and concurrent reuse tests are absent. |
| Family-wide revocation | partial | Clearing the one per-user refresh row and bumping `TokenVersion` revokes the available session, but there are no independent families/devices. |
| Password-change revocation | absent | `ChangePassword` updates only the password hash; it does not close sessions or bump token version. |
| Password-reset revocation | implemented and verified locally | `ResetPassword` calls session closure; auth/full tests passed. |
| Account-state revocation | implemented and verified locally | Active-status and token-version checks reject deactivated/deleted sessions; lifecycle tests passed. |
| Absolute expiry | implemented and verified locally | Absolute expiry is stored and enforced on refresh; auth tests passed. |
| Idle expiry | implemented and verified locally | Idle expiry is checked and capped by absolute expiry; auth tests passed. |
| Cleanup and retention | partial | Expiry fields support TTL-style cleanup, but no dedicated refresh cleanup/retention job or audit evidence was found. |
| Crash recovery | blocked | No crash/retry/conditional-write recovery test. |
| Leakage/logging controls | partial | Refresh secrets are returned to the client and hashes are stored; request logging omits bodies, but there is no comprehensive token-redaction regression suite. Dev mail/SMS adapters can expose related verification material. |

## Required remediation tests

Add a conditional refresh rotation test with two concurrent requests using one token; assert exactly one success, the losing request is treated as reuse, and the entire family/session is revoked. Add tests for password-change invalidation, account-state invalidation, crash/retry recovery, multi-device policy, cleanup retention, and log redaction. These tests are not present in the current source.
