#!/usr/bin/env bash
set -euo pipefail

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
input_file=${AVIA_IPV6_TG_INPUTS_FILE:-}

# Only the committed synthetic fixture may bypass the owner overlay. It is
# local-state-only and cannot be used as a remote input path.
if [[ -n "$input_file" && -f "$input_file" ]] &&
  [[ "$input_file" == "$repository_root/infra/terragrunt/fixtures/"* ]] &&
  rg --quiet --multiline '^\s*fixture_mode\s*=\s*true\s*$' "$input_file"; then
  echo "verified locally: synthetic IPv6 trial fixture; owner and remote actions remain unavailable"
  exit 0
fi

bash "$repository_root/scripts/check-aws-ipv6-trial-decisions.sh"
decision_file=${AVIA_AWS_IPV6_TRIAL_DECISION_FILE:-"$repository_root/.local/aviasurveil360/aws-ipv6-trial/decision.json"}
for variable in AVIA_AWS_IPV6_TRIAL_ACCOUNT_ID AVIA_AWS_IPV6_TRIAL_REGION AVIA_AWS_IPV6_TRIAL_AVAILABILITY_ZONE AVIA_AWS_IPV6_TRIAL_CLOUDFLARE_ACCOUNT_ID AVIA_AWS_IPV6_TRIAL_CLOUDFLARE_ZONE_ID AVIA_AWS_IPV6_TRIAL_CLOUDFLARE_HOSTNAME; do
  [[ -n "${!variable:-}" ]] || { echo "missing-owner-input: $variable must bind the validated decision" >&2; exit 64; }
done
node -e '
  const fs = require("node:fs");
  const [file, account, region, availabilityZone, cloudflareAccount, cloudflareZone, hostname] = process.argv.slice(1);
  const decision = JSON.parse(fs.readFileSync(file, "utf8"));
  const expected = [decision.aws.accountId, decision.aws.region, decision.aws.availabilityZone, decision.cloudflare.accountId, decision.cloudflare.zoneId, decision.cloudflare.hostname];
  const actual = [account, region, availabilityZone, cloudflareAccount, cloudflareZone, hostname];
  if (expected.some((value, index) => value !== actual[index])) {
    console.error("owner-input-binding: Terragrunt environment does not match the validated decision");
    process.exit(65);
  }
' "$decision_file" \
  "$AVIA_AWS_IPV6_TRIAL_ACCOUNT_ID" \
  "$AVIA_AWS_IPV6_TRIAL_REGION" \
  "$AVIA_AWS_IPV6_TRIAL_AVAILABILITY_ZONE" \
  "$AVIA_AWS_IPV6_TRIAL_CLOUDFLARE_ACCOUNT_ID" \
  "$AVIA_AWS_IPV6_TRIAL_CLOUDFLARE_ZONE_ID" \
  "$AVIA_AWS_IPV6_TRIAL_CLOUDFLARE_HOSTNAME"
echo "verified locally: validated decision is bound to the Terragrunt owner inputs"
