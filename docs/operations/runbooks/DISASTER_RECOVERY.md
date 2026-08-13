# Local Disaster Recovery Boundary

The maintained repository has component-level same-host recovery checks for the
application and first-party auth databases. A coordinated full-stack disaster
recovery drill is `not run`. The result remains `candidate-only` and
`release pending`.

## Available local checks

```bash
./scripts/test-backup-profile.sh
./scripts/test-auth-candidate-backup-restore.sh
./scripts/verify-backup-catalog.sh
```

These checks may establish exact local component/catalog integrity and
first-party auth restoration. They do not establish full application/object
recovery, browser recovery, a separate failure domain, host-loss recovery, or
measured RPO/RTO.

## Safety boundary

- Never use an active project or state directory as a destructive target.
- Never mount or overwrite source database/object volumes in a restore target.
- Restrict cleanup to validated task-owned Compose labels and exact temporary
  directories.
- Never use broad Docker prune or broad process-kill commands.
- A partial result is not `verified locally`.

## Required future drill

Before any broader recovery claim, a separately authorized maintained harness
must restore application PostgreSQL, first-party auth PostgreSQL, exact object
versions, and configuration references into a new target; compare exact
fingerprints; verify worker and browser behavior; measure separate database and
object RPO plus RTO; test corrupt-catalog fallback; and prove zero residue.

Until that exists and runs, coordinated DR is `not run`. Remote/account/
region recovery, production identities, retention changes, and real dependency
destruction require separate explicit authorization.
