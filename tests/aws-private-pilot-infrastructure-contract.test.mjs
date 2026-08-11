import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const moduleDirectory = path.join(root, "infra/terraform/modules/aws-private-pilot");

function moduleSource() {
  return readdirSync(moduleDirectory)
    .filter((name) => name.endsWith(".tf"))
    .sort()
    .map((name) => readFileSync(path.join(moduleDirectory, name), "utf8"))
    .join("\n");
}

function occurrences(source, expression) {
  return [...source.matchAll(expression)].length;
}

function validate(source) {
  const errors = [];
  const require = (expression, code) => {
    if (!expression.test(source)) errors.push(code);
  };
  const forbid = (expression, code) => {
    if (expression.test(source)) errors.push(code);
  };

  if (occurrences(source, /resource\s+"aws_instance"\s+"/gu) !== 1) errors.push("exactly-one-ec2");
  if (occurrences(source, /resource\s+"aws_nat_gateway"\s+"/gu) !== 0) errors.push("no-nat-gateway");
  if (occurrences(source, /resource\s+"aws_internet_gateway"\s+"/gu) !== 0) errors.push("no-internet-gateway");
  if (occurrences(source, /resource\s+"aws_egress_only_internet_gateway"\s+"/gu) !== 1) errors.push("exactly-one-egress-only-igw");
  if (occurrences(source, /resource\s+"aws_eip"\s+"/gu) !== 0) errors.push("no-eip");
  if (occurrences(source, /resource\s+"aws_lb"\s+"/gu) !== 0) errors.push("no-alb");
  if (occurrences(source, /resource\s+"aws_db_instance"\s+"/gu) !== 1) errors.push("exactly-one-rds");
  if (occurrences(source, /resource\s+"aws_vpc_endpoint"\s+"/gu) !== 1) errors.push("s3-gateway-only");

  require(/condition\s*=\s*var\.instance_type\s*==\s*"t4g\.small"/u, "t4g-small-only");
  require(/Architecture\s*=\s*"linux-arm64"/u, "arm64-only");
  require(/instance_class\s*=\s*"db\.t4g\.micro"/u, "db-t4g-micro-only");
  require(/publicly_accessible\s*=\s*false\s+multi_az\s*=\s*false/u, "single-az-rds");
  require(/publicly_accessible\s*=\s*false/u, "private-rds");
  require(/backup_retention_period\s*=\s*14/u, "rds-pitr-14-days");
  require(/availability_zone\s*=\s*var\.availability_zones\[0\]/u, "rds-in-workload-az-a");
  require(/check\s+"availability_zones_belong_to_region"/u, "availability-zone-region-binding");
  require(/condition\s*=\s*var\.kms_alias_prefix\s*==\s*"aviasurveil360-private-pilot"/u, "project-kms-alias-binding");

  require(/assign_generated_ipv6_cidr_block\s*=\s*true/u, "vpc-ipv6-required");
  require(/resource\s+"aws_subnet"\s+"dual_stack_app_a"[\s\S]*?assign_ipv6_address_on_creation\s*=\s*true/u, "dual-stack-app-subnet");
  require(/ipv6_cidr_block\s*=\s*"::\/0"[\s\S]*?egress_only_gateway_id/u, "egress-only-ipv6-route");
  forbid(/cidr_block\s*=\s*"0\.0\.0\.0\/0"/u, "ipv4-default-route");
  require(/resource\s+"aws_vpc_security_group_ingress_rule"\s+"database_application"/u, "database-ingress-only");
  forbid(/resource\s+"aws_vpc_security_group_ingress_rule"\s+"application_/u, "runtime-ingress-forbidden");
  require(/for_each\s*=\s*local\.cloudflare_tunnel_egress_rules[\s\S]*?from_port\s*=\s*7844[\s\S]*?to_port\s*=\s*7844/u, "cloudflare-ipv6-7844-egress");
  require(/cloudflare_tunnel_ipv6_cidrs[\s\S]*?can\(cidrhost\(cidr, 0\)\)/u, "cloudflare-ipv6-cidr-validation");
  require(/for_each\s*=\s*var\.smtp_ipv6_cidrs/u, "smtp-ipv6-egress");
  forbid(/data_feed_ipv6_cidrs|application_data_feed/u, "no-data-feed-egress");
  require(/cidr_ipv4\s*=\s*"\$\{cidrhost\(var\.vpc_cidr, 2\)\}\/32"/u, "exact-vpc-resolver-egress");
  require(/resource\s+"cloudflare_zero_trust_tunnel_cloudflared"/u, "cloudflare-tunnel-required");
  require(/resource\s+"cloudflare_zero_trust_tunnel_cloudflared"\s+"this"[\s\S]*?lifecycle\s*\{\s*prevent_destroy\s*=\s*true/u, "cloudflare-tunnel-destroy-guard");
  require(/resource\s+"cloudflare_zero_trust_tunnel_cloudflared_config"[\s\S]*?service\s*=\s*"http:\/\/127\.0\.0\.1:8080"[\s\S]*?service\s*=\s*"http_status:404"/u, "loopback-tunnel-ingress");
  require(/resource\s+"cloudflare_dns_record"\s+"application"[\s\S]*?for_each\s*=\s*var\.cloudflare_dns_cutover_authorized\s*\?\s*toset\(\[var\.hostname\]\)\s*:\s*toset\(\[\]\)/u, "cloudflare-dns-separate-cutover");
  require(/proxied\s*=\s*true/u, "cloudflare-proxy-required");
  require(/check\s+"cloudflare_dns_is_a_separate_cutover"/u, "cloudflare-dns-cutover-check");
  require(/resource\s+"aws_ssm_parameter"\s+"cloudflare_connector"[\s\S]*?type\s*=\s*"SecureString"[\s\S]*?value_wo\s*=\s*"PENDING_SEPARATE_AUTHORIZATION"[\s\S]*?value_wo_version\s*=\s*1/u, "write-only-connector-placeholder");
  forbid(/data\s+"cloudflare_zero_trust_tunnel_cloudflared_token"/u, "connector-token-in-terraform-state");

  require(/vpc_endpoint_type\s*=\s*"Gateway"/u, "s3-gateway-endpoint");
  require(/PullEcrLayersThroughGatewayEndpoint[\s\S]*?prod-\$\{var\.region\}-starport-layer-bucket/u, "ecr-layer-gateway-access");
  require(/object_buckets\s*=\s*toset\(\[\s*"quarantine",\s*"canonical",\s*"attachments",\s*"documents"/u, "four-private-buckets");
  require(/resource\s+"aws_s3_bucket_versioning"/u, "s3-versioning");
  require(/sse_algorithm\s*=\s*"aws:kms"/u, "s3-kms-encryption");
  require(/restrict_public_buckets\s*=\s*true/u, "s3-public-block");
  require(/resource\s+"aws_s3_bucket_cors_configuration"[\s\S]*?allowed_origins\s*=\s*\["https:\/\/\$\{var\.hostname\}"\]/u, "exact-browser-cors-origin");
  require(/resource\s+"aws_guardduty_malware_protection_plan"/u, "guardduty-plan");
  require(/status\s*=\s*"ENABLED"/u, "guardduty-result-tagging");
  require(/NO_THREATS_FOUND/u, "guardduty-clean-result");
  require(/GetObjectVersionTagging/u, "exact-version-result-read");
  require(/WriteGuardDutyValidationObject[\s\S]*?malware-protection-resource-validation-object/u, "guardduty-validation-object");
  require(/ManageGuardDutyOwnedEventRule[\s\S]*?events:ManagedBy/u, "guardduty-managed-rule-condition");
  require(/InspectGuardDutyOwnedEventRule[\s\S]*?events:ListTargetsByRule/u, "guardduty-managed-rule-inspection");
  require(/DenyRuntimeScanTagMutation/u, "scan-tag-tamper-denial");
  require(/DenyOrdinaryQuarantineReadUntilExactVersionIsClean[\s\S]*?ArnNotEquals/u, "non-clean-ordinary-read-denial");
  require(/DenyOrdinaryQuarantineReadUntilExactVersionIsClean[\s\S]*?ArnNotEquals[\s\S]*?aws_iam_role\.guardduty_malware\.arn/u, "guardduty-only-quarantine-read-exception");
  forbid(/ArnNotEquals[\s\S]{0,200}?aws_iam_role\.runtime\.arn/u, "runtime-must-not-bypass-clean-tag");

  require(/resource\s+"aws_iam_instance_profile"/u, "instance-profile");
  require(/http_put_response_hop_limit\s*=\s*2/u, "imds-v2-docker-provider-chain");
  require(/http_protocol_ipv6\s*=\s*"enabled"/u, "imds-ipv6");
  require(/ipv6_address_count\s*=\s*1/u, "one-global-ipv6");
  forbid(/s3:ListBucketVersions/u, "unneeded-runtime-bucket-listing");
  require(/--setopt=ip_resolve=6/u, "bootstrap-forces-ipv6");
  require(/amazon-ecr-credential-helper/u, "ecr-instance-profile-credential-helper");
  require(/docker-compose-plugin[\s\S]*?docker compose version/u, "compose-plugin-preflight");
  require(/"credHelpers"[\s\S]*?"ecr-login"/u, "ecr-helper-configuration");
  require(/\.dkr-ecr\.\$\{var\.region\}\.on\.aws/u, "ipv6-ecr-registry");
  require(/"ipv6": true/u, "docker-ipv6-enabled");
  require(/"ip6tables": true/u, "docker-ipv6-filtering");
  require(/"Agent": \{\s*"Region": "\$\{var\.region\}",\s*"UseDualStackEndpoint": true/u, "ssm-dual-stack");
  require(/"use_dualstack_endpoint": true/u, "cloudwatch-dual-stack");
  require(/\/etc\/aviasurveil360\/private-pilot\/docker\/config\.json/u, "protected-ecr-helper-configuration");
  forbid(/\/root\/\.docker/u, "systemd-inaccessible-ecr-helper-configuration");
  require(/ReadRuntimeSecretReferences/u, "scoped-secret-access");
  require(/ReadCloudflareConnectorParameter[\s\S]*?ssm:GetParameter/u, "scoped-connector-parameter-access");
  require(/AmazonSSMManagedInstanceCore/u, "ssm-session-management");
  require(/image_tag_mutability\s*=\s*"IMMUTABLE"/u, "immutable-ecr");
  require(/resource\s+"aws_ecr_repository"\s+"runtime"[\s\S]*?lifecycle\s*\{\s*prevent_destroy\s*=\s*true/u, "ecr-destroy-guard");
  require(/scan_on_push\s*=\s*true/u, "ecr-scan-on-push");
  require(/resource\s+"aws_cloudwatch_log_group"/u, "cloudwatch-required");
  require(/resource\s+"aws_sns_topic"\s+"alerts"[\s\S]*?kms_master_key_id\s*=\s*"alias\/aws\/sns"/u, "encrypted-alert-topic-required");
  require(/CloudWatchAndBudgetsPublish[\s\S]*?budgets\.amazonaws\.com[\s\S]*?cloudwatch\.amazonaws\.com/u, "alert-topic-service-policy");
  require(/CloudflaredTunnelHAConnections/u, "tunnel-health-alarm");
  require(/threshold\s*=\s*4/u, "four-tunnel-connections");
  require(/retention_in_days\s*=\s*30/u, "cloudwatch-bounded-retention");
  require(/amazon-cloudwatch-agent-ctl[\s\S]*?-a fetch-config[\s\S]*?-m ec2[\s\S]*?-s/u, "cloudwatch-agent-started");
  require(/\/var\/lib\/docker\/containers\/\*\/\*\.log/u, "container-logs-exported");
  require(/AviaSurveil360\/PrivatePilot/u, "host-metric-namespace");
  require(/mem_used_percent/u, "host-memory-metric");
  require(/swap_used_percent/u, "host-swap-metric");
  require(/used_percent/u, "host-disk-metric");
  require(/resource\s+"aws_backup_plan"/u, "aws-backup-required");
  require(/delete_after\s*=\s*35/u, "backup-35-days");
  require(/AWSBackupServiceRolePolicyForS3Backup/u, "s3-backup-permissions");
  require(/AWSBackupServiceRolePolicyForS3Restore/u, "s3-restore-permissions");
  require(/resource\s+"aws_budgets_budget"/u, "budget-required");
  require(/subscriber_sns_topic_arns\s*=\s*\[aws_sns_topic\.alerts\.arn\]/u, "budget-sns-notification-required");
  forbid(/resource\s+"aws_budgets_budget"[\s\S]*?cost_filter\s*\{/u, "account-wide-budget-required");

  forbid(/resource\s+"aws_(?:autoscaling_group|launch_template|wafv2_[^"]+|ses_[^"]+|rds_cluster|iam_access_key|secretsmanager_secret_version|nat_gateway|internet_gateway|eip|lb(?:_[^"]+)?)"/u, "forbidden-resource");
  forbid(/vpc_endpoint_type\s*=\s*"Interface"/u, "interface-endpoint-sprawl");
  forbid(/associate_public_ip_address\s*=\s*true/u, "public-ec2");
  forbid(/(?<!_)access_key\s*=|(?<!_)secret_key\s*=/u, "static-aws-credentials");
  forbid(/docker login/u, "stored-ecr-login-token");
  forbid(/Action\s*=\s*"\*"/u, "wildcard-iam-action");

  if (occurrences(source, /Resource\s*=\s*"\*"/gu) !== 4) errors.push("bounded-wildcard-resources");
  require(/ObtainEcrAuthorizationToken[\s\S]*?Resource\s*=\s*"\*"/u, "ecr-token-wildcard-only");
  require(/PublishPrivatePilotMetrics[\s\S]*?cloudwatch:namespace/u, "metric-wildcard-conditioned");
  require(/AccountKeyAdministration[\s\S]*?Resource\s*=\s*"\*"/u, "kms-root-administration-only");
  require(/CloudWatchLogsEncryption[\s\S]*?kms:EncryptionContext:aws:logs:arn/u, "kms-log-wildcard-conditioned");

  return errors;
}

test("private-pilot module encodes the exact cost-bounded topology", () => {
  assert.deepEqual(validate(moduleSource()), []);
});

test("unsafe infrastructure mutations are rejected offline", async (t) => {
  const source = moduleSource();
  const mutations = {
    "second-ec2": [source + '\nresource "aws_instance" "second" {}', "exactly-one-ec2"],
    nat: [source + '\nresource "aws_nat_gateway" "forbidden" {}', "no-nat-gateway"],
    igw: [source + '\nresource "aws_internet_gateway" "forbidden" {}', "no-internet-gateway"],
    alb: [source + '\nresource "aws_lb" "forbidden" {}', "no-alb"],
    eip: [source + '\nresource "aws_eip" "forbidden" {}', "no-eip"],
    autoscaling: [source + '\nresource "aws_autoscaling_group" "fleet" {}', "forbidden-resource"],
    "multi-az-rds": [source.replace(/multi_az\s*=\s*false/u, "multi_az = true"), "single-az-rds"],
    "public-rds": [source.replace(/publicly_accessible\s*=\s*false/u, "publicly_accessible = true"), "private-rds"],
    "public-ec2": [source.replace(/associate_public_ip_address\s*=\s*false/u, "associate_public_ip_address = true"), "public-ec2"],
    "ipv4-default": [source + '\nresource "aws_route" "forbidden" { destination_cidr_block = "0.0.0.0/0" }', "ipv4-default-route"],
    "interface-endpoint": [source.replace(/vpc_endpoint_type\s*=\s*"Gateway"/u, 'vpc_endpoint_type = "Interface"'), "s3-gateway-endpoint"],
    "missing-guardduty-tag": [source.replace(/status\s*=\s*"ENABLED"/u, 'status = "DISABLED"'), "guardduty-result-tagging"],
    "mutable-ecr": [source.replace(/image_tag_mutability\s*=\s*"IMMUTABLE"/u, 'image_tag_mutability = "MUTABLE"'), "immutable-ecr"],
    "static-key": [source + '\nprovider "aws" { access_key = "fixture" }', "static-aws-credentials"],
    "connector-token-data": [source + '\ndata "cloudflare_zero_trust_tunnel_cloudflared_token" "forbidden" {}', "connector-token-in-terraform-state"],
    waf: [source + '\nresource "aws_wafv2_web_acl" "creep" {}', "forbidden-resource"],
    ses: [source + '\nresource "aws_ses_domain_identity" "creep" {}', "forbidden-resource"],
  };
  for (const [name, [mutated, expected]] of Object.entries(mutations)) {
    await t.test(name, () => assert.ok(validate(mutated).includes(expected), validate(mutated).join(", ")));
  }
});

test("Terragrunt remains input-only, backend-disabled, and non-deployable by default", () => {
  const environment = path.join(root, "infra/terragrunt/environments/aws-private-pilot");
  const rootConfig = readFileSync(path.join(environment, "root.hcl"), "utf8");
  const example = readFileSync(path.join(environment, "region.hcl.example"), "utf8");
  const fixture = readFileSync(path.join(root, "infra/terragrunt/fixtures/aws-private-pilot-non-deployable.hcl"), "utf8");
  const stack = readFileSync(path.join(environment, "components/stack/terragrunt.hcl"), "utf8");
  const bootstrap = readFileSync(path.join(environment, "bootstrap/remote-state/terragrunt.hcl"), "utf8");
  const managedBootstrap = readFileSync(path.join(environment, "bootstrap/remote-state-managed/terragrunt.hcl"), "utf8");

  assert.match(rootConfig, /AVIA_AWS_PRIVATE_PILOT_TG_INPUTS_FILE/u);
  assert.match(rootConfig, /AVIA_TG_DISABLE_BACKEND", "true"/u);
  assert.match(rootConfig, /AVIA_AWS_PRIVATE_PILOT_FAST_READ_ONLY_PLAN/u);
  assert.match(rootConfig, /hook_commands/u);
  assert.match(rootConfig, /\["plan", "apply", "destroy"\]/u);
  assert.match(rootConfig, /\["apply", "destroy"\]/u);
  assert.match(rootConfig, /remote-hook/u);
  assert.match(rootConfig, /aws_profile\s*=\s*"avia"/u);
  assert.match(rootConfig, /profile\s*=\s*local\.aws_profile/u);
  assert.match(example, /remote_actions_authorized\s*=\s*false/u);
  assert.match(example, /aws_profile\s*=\s*"avia"/u);
  assert.match(example, /AVIA_AWS_PRIVATE_PILOT_REGION", "eu-central-1"/u);
  assert.match(example, /cloudflare_tunnel_ipv6_cidrs/u);
  assert.match(example, /smtp_ipv6_cidrs/u);
  assert.doesNotMatch(example, /public_alb|certificate_arn|origin_header|smtp_ipv4/u);
  assert.doesNotMatch(example, /aws_profile\s*=\s*"default"/u);
  assert.doesNotMatch(example, /\b\d{12}\b/u);
  assert.match(fixture, /fixture_mode\s*=\s*true/u);
  assert.match(fixture, /remote_actions_authorized\s*=\s*false/u);
  assert.match(fixture, /aws_profile\s*=\s*"avia"/u);
  assert.match(fixture, /fixture\.example\.invalid/u);
  assert.match(stack, /modules\/aws-private-pilot/u);
  assert.match(stack, /profile\s*=\s*"\$\{include\.root\.locals\.aws_profile\}"/u);
  assert.doesNotMatch(stack, /access_key|secret_key/u);
  assert.match(bootstrap, /bootstrap\/remote-state/u);
  assert.match(bootstrap, /profile\s*=\s*"\$\{local\.aws_profile\}"/u);
  assert.match(bootstrap, /allowed_account_ids\s*=\s*\["\$\{local\.account_id\}"\]/u);
  assert.match(bootstrap, /bootstrap-plan-hook/u);
  assert.match(bootstrap, /commands\s*=\s*\["apply", "destroy", "import"\][\s\S]*?remote-hook/u);
  assert.doesNotMatch(bootstrap, /access_key|secret_key/u);
  assert.match(managedBootstrap, /AVIA_AWS_PRIVATE_PILOT_BOOTSTRAP_MANAGED_INPUTS_FILE/u);
  assert.match(managedBootstrap, /backend\s*=\s*"s3"/u);
  assert.match(managedBootstrap, /disable_init\s*=\s*false/u);
  assert.match(managedBootstrap, /key\s*=\s*"aws-private-pilot\/bootstrap\/remote-state\/terraform\.tfstate"/u);
  assert.match(managedBootstrap, /use_lockfile\s*=\s*true/u);
  assert.match(managedBootstrap, /allowed_account_ids\s*=\s*\[local\.account_id\]/u);
  assert.match(managedBootstrap, /profile\s*=\s*local\.aws_profile/u);
  assert.match(managedBootstrap, /commands\s*=\s*\["plan", "apply", "destroy", "import"\][\s\S]*?remote-hook/u);
  assert.doesNotMatch(managedBootstrap, /access_key|secret_key/u);
});

test("remote hooks reject default or omitted AWS profiles before authorization", () => {
  const checker = path.join(root, "scripts/check-aws-private-pilot-infrastructure.sh");
  const omitted = spawnSync(checker, ["remote-hook"], {
    cwd: root,
    encoding: "utf8",
    env: { ...process.env, AWS_PROFILE: "" },
  });
  assert.equal(omitted.status, 76);
  assert.match(omitted.stderr, /aws-operator-profile-must-be-avia/u);

  const defaultProfile = spawnSync(checker, ["remote-hook"], {
    cwd: root,
    encoding: "utf8",
    env: { ...process.env, AWS_PROFILE: "default" },
  });
  assert.equal(defaultProfile.status, 76);
  assert.match(defaultProfile.stderr, /default and omitted AWS profiles are forbidden/u);

  const avia = spawnSync(checker, ["remote-hook"], {
    cwd: root,
    encoding: "utf8",
    env: { ...process.env, AWS_PROFILE: "avia" },
  });
  assert.equal(avia.status, 77);
  assert.match(avia.stderr, /remote-action-unauthorized/u);
});

test("bootstrap plan hook requires the exact no-apply scope, avia, and eu-central-1", () => {
  const checker = path.join(root, "scripts/check-aws-private-pilot-infrastructure.sh");
  const base = {
    ...process.env,
    AWS_PROFILE: "avia",
    AWS_REGION: "eu-central-1",
    AWS_DEFAULT_REGION: "eu-central-1",
    TG_TF_PATH: "terraform",
  };
  const missing = spawnSync(checker, ["bootstrap-plan-hook"], { cwd: root, encoding: "utf8", env: base });
  assert.equal(missing.status, 77);
  assert.match(missing.stderr, /exact bootstrap provider-plan authorization/u);

  const wrongRegion = spawnSync(checker, ["bootstrap-plan-hook"], {
    cwd: root,
    encoding: "utf8",
    env: { ...base, AWS_REGION: "us-east-1", AVIA_AWS_PRIVATE_PILOT_PLAN_AUTHORIZATION: "remote-state-bootstrap-provider-plan-no-apply" },
  });
  assert.equal(wrongRegion.status, 76);
  assert.match(wrongRegion.stderr, /aws-region-must-be-eu-central-1/u);

  const wrongCli = spawnSync(checker, ["bootstrap-plan-hook"], {
    cwd: root,
    encoding: "utf8",
    env: { ...base, TG_TF_PATH: "tofu", AVIA_AWS_PRIVATE_PILOT_PLAN_AUTHORIZATION: "remote-state-bootstrap-provider-plan-no-apply" },
  });
  assert.equal(wrongCli.status, 76);
  assert.match(wrongCli.stderr, /terraform-cli-must-be-selected-explicitly/u);

  const authorized = spawnSync(checker, ["bootstrap-plan-hook"], {
    cwd: root,
    encoding: "utf8",
    env: { ...base, AVIA_AWS_PRIVATE_PILOT_PLAN_AUTHORIZATION: "remote-state-bootstrap-provider-plan-no-apply" },
  });
  assert.equal(authorized.status, 0, authorized.stderr);
  assert.match(authorized.stdout, /provider plan only; apply remains blocked/u);
});

test("OPA source covers cost and security mutation boundaries", () => {
  const policy = readFileSync(path.join(root, "infra/policies/aws-private-pilot.rego"), "utf8");
  const mutations = readFileSync(path.join(root, "infra/policies/aws-private-pilot_test.rego"), "utf8");

  for (const boundary of [
    "exactly one EC2 instance",
    "must not contain a NAT Gateway",
    "exactly one egress-only Internet Gateway",
    "exactly one RDS instance",
    "exactly one S3 Gateway Endpoint",
    "db.t4g.micro",
    "aws_guardduty_malware_protection_plan",
    "aws_budgets_budget",
    "cloudflare_dns_record",
    "cloudflare_zero_trust_tunnel_cloudflared",
    "wildcard IAM",
  ]) {
    assert.match(policy, new RegExp(boundary.replaceAll(".", "\\."), "u"));
  }
  assert.match(mutations, /test_second_compute_is_denied/u);
  assert.match(mutations, /test_multi_az_rds_is_denied/u);
  assert.match(mutations, /test_interface_endpoint_is_denied/u);
  assert.match(mutations, /test_secret_value_resource_is_denied/u);
});
