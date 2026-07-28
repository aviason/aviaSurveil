# Remote-State Bootstrap And Migration

This procedure is `candidate-only`. It does not authorize an AWS plan, change,
or state migration.

1. Record the approved account, region/data-residency decision, state bucket
   name, KMS alias, owners, budget, change window, and retention decision in the
   untracked owner input file.
2. Run the bootstrap unit alone and preserve its plan hash. The bootstrap unit
   is the only phase allowed to use temporary local state.
3. Obtain separate authorization for the exact bootstrap plan and wrapper
   hashes before any AWS change.
4. After the authorized bootstrap phase, verify the returned bucket, KMS key,
   versioning, public-access block, and native `use_lockfile` contract.
5. Update the untracked owner input with the exact bootstrap outputs. Render
   each later unit and confirm its generated S3 backend key is phase-scoped.
6. For one reviewed unit at a time, preserve the current local state hash and
   run Terraform initialization with `-migrate-state`. Stop on any backend,
   caller, region, KMS, lock, or state-lineage mismatch.
7. Verify the encrypted remote state, lockfile behavior, local-state cleanup
   record, and manifest hash before proceeding to the next unit.

Never migrate all units concurrently. Never delete local state until its remote
lineage and encrypted retained copy have been independently verified.
