locals {
  # Synthetic values are for offline Terraform/Terragrunt shape tests only.
  # This fixture is not an owner decision and cannot authorize a provider call.
  fixture_mode                        = true
  account_id                          = "111122223333"
  region                              = "eu-central-1"
  availability_zone                   = "eu-central-1a"
  state_bucket_name                   = "avia-ipv6-fixture-state-111122223333"
  state_kms_key_arn                   = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
  ecr_kms_key_arn                     = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
  runtime_kms_key_arn                 = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
  vpc_ipv4_cidr                       = "10.76.0.0/16"
  cloudflare_account_id               = "cf-account-fixture"
  cloudflare_zone_id                  = "cf-zone-fixture"
  cloudflare_hostname                 = "demo.aviacaa.example"
  cloudflare_connector_parameter_name = "/avia-ipv6-trial/cloudflare-connector"
  cloudflare_access_public_exposure   = true
  cloudflare_allowed_identities       = []
  cloudflare_allowed_domains          = []
  cloudflare_tunnel_ipv6_cidrs        = ["2606:4700::/32"]
  management_ipv6_cidrs               = ["2600:1f18::/32"]
  dns_ipv6_cidrs                      = ["2600:1f18:ffff::/48"]
  bootstrap_https_ipv6_cidrs          = ["2600:1f18:eeee::/48"]
  image_uris = {
    gateway    = "111122223333.dkr-ecr.eu-central-1.on.aws/gateway@sha256:0000000000000000000000000000000000000000000000000000000000000000"
    "web-demo" = "111122223333.dkr-ecr.eu-central-1.on.aws/web-demo@sha256:1111111111111111111111111111111111111111111111111111111111111111"
  }
  cloudflared_image                 = "cloudflare/cloudflared@sha256:2222222222222222222222222222222222222222222222222222222222222222"
  ami_ssm_parameter_name            = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64"
  ami_id                            = "ami-0123456789abcdef0"
  root_volume_size_gib              = 20
  delete_root_volume_on_termination = true
  ecr_registry_host                 = "111122223333.dkr-ecr.eu-central-1.on.aws"
  compose_bundle_path               = "/opt/aviasurveil360/aws-ipv6-trial/compose.yaml"
  compose_bundle_sha256             = "3333333333333333333333333333333333333333333333333333333333333333"
  monthly_ceiling_usd               = 25
  one_run_ceiling_usd               = 10
  estimated_monthly_usd             = 12
  estimated_one_run_usd             = 5
  trial_expiry                      = "2026-12-31T23:59:59Z"
  alert_recipients                  = ["cost-owner@example.invalid"]
  tags = {
    Environment        = "fixture"
    Owner              = "platform-operations"
    CostCenter         = "trial-001"
    DataClassification = "restricted"
    ManagedBy          = "terraform"
  }
}
