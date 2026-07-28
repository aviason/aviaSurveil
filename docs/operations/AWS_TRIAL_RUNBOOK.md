# AWS Trial Plan And Command Runbook

**Status:** `candidate-only`; Task 10 AWS execution is `not run`.

**Owners:** Platform operates the commands; Security reviews policy and image
evidence; Records approves region and retention; Release authority grants each
exact action. No single owner may infer another owner's decision.

## Preconditions

1. Complete local reliability, recovery, Terraform, and Terragrunt gates.
2. Create the owner-approved untracked decision described in
   `AWS_TRIAL_DECISIONS.md`.
3. Use a dedicated trial account and read-only planner role for discovery and
   planning. Confirm the account and region before any command.
4. Set immutable image, CycloneDX SBOM, zero HIGH/CRITICAL Trivy, provider lock,
   and bounded cost evidence.
5. Keep `.local/aviasurveil360/aws-plans/` at mode `0700`; files are `0600`.

## Plan One Phase

Run `scripts/aws-trial-plan.sh preview <phase>` first. The preview contacts no
AWS service. A real plan requires the exact `plan` authorization printed from
the reviewed decision hash. Execute only inside the approved window:

```bash
scripts/aws-trial-plan.sh execute <phase> <decision.json> <bundle-directory>
```

The wrapper resolves the current caller, plans only units in that phase,
creates separate binary/JSON artifacts, records cost and image evidence, runs
`check-aws-plan.sh`, and stops. It performs no apply.

Review the diff, OPA result, cost/capacity, public exposure, data protection,
caller/account/region, image digest/SBOM/scan, wrapper hashes, expiry, and
cleanup record. Any change requires a new bundle and review.

## Apply And Publish

Request a new exact authorization for one action and one phase. Then use:

```bash
scripts/aws-trial-apply.sh execute <phase> <bundle-directory>
scripts/aws-trial-publish-artifacts.sh execute artifact-publication <bundle-directory>
```

Bootstrap may create only state KMS/S3 resources. `foundation-ecr` precedes
artifact publication. `data-runtime` requires a newly planned bundle after
publication. Stop between phases. Never use broad `run --all apply`.

The untracked publication manifest contains one pipe-delimited row per image:
repository, expected digest, image archive, CycloneDX SBOM path, expected SBOM
SHA-256, and the loaded source image reference. The wrapper recomputes the SBOM
hash, publishes a digest-derived immutable tag to the reviewed account/region,
and reads the ECR digest back before accepting publication.

## Smoke

After an authorized `data-runtime` apply, run:

```bash
scripts/aws-trial-smoke.sh execute data-runtime <bundle-directory>
```

The frozen smoke covers HTTPS/security headers, OIDC and MFA, all 86 routes,
canonical mutation authority, private Evidence scanning, private trial
Mailpit, PDF generation, telemetry/alerts, and backup status. A failed health,
authority, privacy, digest, cost, or alert check stops the trial.

## Rollback

Create and review a plan that changes compute to the previous immutable digest
without changing databases, buckets, backups, or state. Bind that plan and
digest to a fresh authorization, run `aws-trial-rollback.sh`, then repeat smoke.
Do not use database deletion as application rollback.

## Retain Or Destroy

Follow the decision file. Retention requires named age recipients, encrypted
artifacts, expiry, and cleanup ownership. Destroy requires a fresh,
phase-scoped destroy plan plus an exact Terraform-state resource manifest and a
tagged-resource inventory. Run `aws-trial-destroy.sh` only after exact destroy
authorization. Never perform account-wide deletion.

After destroy, query the approved account/region for the exact stack tags,
check ECR, EC2/EBS/ELB, RDS/snapshots, S3, KMS, Secrets Manager, CloudWatch,
Backup, and remote-state disposition, and record zero unintended billable
residue. Preserve required audit evidence before cleanup.

## Stop Conditions

Stop immediately for caller/account/region mismatch, stale or changed plan,
missing owner decision, cost/capacity breach, policy denial, public database or
object storage, wildcard IAM, mutable/unscanned image, missing SBOM, unhealthy
service, failed backup, unexpected resource, or absent rollback/destroy scope.
