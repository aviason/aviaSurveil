# AWS IPv6 ARM64 Trial Runbook

**Status:** `candidate-only`; local contracts are `verified locally`; native
ARM64 capacity, AWS/Cloudflare discovery, remote plans, publication, apply,
smoke, retain/destroy, and residue queries are `not run`.

This runbook is for the separate `aws-ipv6-trial` environment only. It does
not authorize an AWS or Cloudflare action and does not change the existing
`aws-trial` or paused AWS preprod topology. Every remote or cost-bearing step
needs a new exact authorization naming the account, region, action, reviewed
artifact/plan hash, budget, expiry, and stop/retention decision.

## Local preconditions

1. Read the [decision contract](AWS_IPV6_TRIAL_DECISIONS.md) and create its
   untracked mode `0600` owner overlay only when the owners are ready to bind
   exact account, region, AZ, Cloudflare, cost, lifecycle, and ownership
   choices. Do not put secret values in the overlay.
2. Run the decision and layout checks. An absent overlay must remain a
   `missing-owner-input` stop:

   ```bash
   ./scripts/check-aws-ipv6-trial-decisions.sh
   ./scripts/check-aws-ipv6-trial-terragrunt.sh
   ```

   A non-fixture Terragrunt plan also requires its account, region, AZ,
   Cloudflare account/zone, and hostname environment inputs to match the
   validated decision exactly; the preflight hook checks that binding.

3. Build only on a native ARM64 host with digest-bound `cloudflared`, gateway,
   and web-demo images. The image script does not pull or publish anything:

   ```bash
   ./scripts/build-aws-ipv6-trial-images.sh \
     --platform linux/arm64 \
     --cloudflared-image <repository>@sha256:<digest>
   ```

4. Use the task-owned runtime fixture only after image evidence exists. The
   default invocation deliberately reports `not run`; `--run` requires a
   root-owned token file and cleans only its own Compose project:

   ```bash
   ./scripts/test-aws-ipv6-trial-runtime.sh
   ./scripts/test-aws-ipv6-trial-runtime.sh --run
   ```

The first runtime is exactly ARM64 `cloudflared`, the internal gateway, and
the ARM64 web demo. It has no host-published application ports, no inbound
security-group rules, no SSH, and no application role/organization authority
in the gateway. Cloudflare edge selection is fixed to
`TUNNEL_EDGE_IP_VERSION=6`.

## Remote stop boundaries

The following are separate authorization points, in dependency order, and
must not be inferred from the local checks:

1. Read-only AWS/Cloudflare discovery and current pricing/capacity evidence.
2. Remote-state bootstrap and lock acquisition.
3. A reviewed, redacted, policy-checked plan for one foundation wave.
4. ARM64 ECR publication after local manifest/SBOM/provenance/scan evidence.
5. Cloudflare Tunnel/DNS/Access and the SecureString connector-parameter
   handoff in its dedicated state boundary.
6. The single IPv6-only `t4g.small` compute wave and SSM-only administration.
7. The bounded smoke/capacity observation window.
8. An exact retain or scoped-destroy action, followed by an exact residue query.

Stop for any owner-input mismatch, stale or changed artifact, cost breach,
policy denial, public IPv4/EIP/NAT/LB/RDS/interface endpoint, inbound rule,
SSH path, amd64/emulation, mutable or unscanned image, secret in state/logs,
failed health/capacity threshold, or unexpected resource. A local or remote
failure remains evidence; it is not rewritten as a passing result.

## Evidence language

Local source and contract checks may be labeled `verified locally`. Image
build, browser loop, native `t4g.small` capacity, remote discovery, provider
initialization, OPA/TFLint/Trivy execution, and all remote actions remain
`not run` until separately authorized and actually performed. The trial stays
`candidate-only`, release is `release pending`, and `production-ready: not established`.
