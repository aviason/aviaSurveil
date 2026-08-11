# Migration boundary

`001_auth_identity.sql` is the only executable export migration. Its lexical
position is its exact execution order. It is a forward-only, current-state
projection of the auth-related portions of source migrations `001`, `006`,
`010`, `011`, `015`, `017`, and `035`; it is not a replacement for those files
inside the source repository.

The projection deliberately keeps `account_status`, even though the copied
auth package does not enforce it. AviaSurveil360 must add fail-closed status and
organization checks before using the package. The projection does not add a
rollback, compatibility view, trigger, password-reset table, MFA table, or
organization table because those capabilities do not exist in the source.

Apply it only to a new task-owned test database. Integration into an existing
schema requires a new AviaSurveil360 migration reviewed against that schema.
