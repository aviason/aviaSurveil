# AviaSurveil360 Certification Boundary

Current state: `candidate-only`; certification is `blocked`.

`make harness-check` and `make harness-maintenance` validate only Surveil's
local documentation and semantic routes. They do not run the candidate runtime,
Workspace integration composition, Auth endpoint qualification, or a Data
native gate. Auth qualification remains `not run` without its named
caller-provided PostgreSQL and Mailpit fixtures.

Do not create source or direct-child attestation commits, evidence signatures,
or HMAC keys. Any future evidence must bind to the exact authorized source commit
and exact child identity; caller-owned external HMAC key custody is required and
this repository must neither generate nor store that key. Data's
`DEBT-001` deliberate harness retirement is an owner-controlled blocker: do
not recreate a Data harness, certification tree, HMAC snapshot, or fixture.

Until explicit source-commit authority, child attestation authority, Auth
fixture evidence, caller-owned external HMAC key custody, and a Data-owner
decision exist, report `blocked`, never certified.
