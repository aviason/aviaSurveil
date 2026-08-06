mock_provider "aws" {}
mock_provider "cloudflare" {}

variables {
  tags = {
    Environment        = "fixture"
    Owner              = "platform-operations"
    CostCenter         = "trial-001"
    DataClassification = "restricted"
    ManagedBy          = "terraform"
  }
}

run "ipv6_trial_network_is_ipv6_native_and_egress_only" {
  command = plan

  module {
    source = "../modules/ipv6-trial-network"
  }

  variables {
    name                         = "avia-ipv6-fixture"
    vpc_ipv4_cidr                = "10.76.0.0/16"
    availability_zone            = "eu-central-1a"
    cloudflare_tunnel_ipv6_cidrs = ["2606:4700::/32"]
    management_ipv6_cidrs        = ["2600:1f18::/32"]
    dns_ipv6_cidrs               = ["2600:1f18:ffff::/48"]
    bootstrap_https_ipv6_cidrs   = ["2600:1f18:eeee::/48"]
  }

  assert {
    condition     = aws_vpc.this.assign_generated_ipv6_cidr_block
    error_message = "The trial VPC must request an AWS-provided IPv6 range."
  }

  assert {
    condition     = aws_subnet.runtime.ipv6_native && !aws_subnet.runtime.map_public_ip_on_launch
    error_message = "The runtime subnet must be IPv6-native without public IPv4 mapping."
  }

  assert {
    condition     = aws_route_table.runtime.route[0].ipv6_cidr_block == "::/0"
    error_message = "The runtime route table must have only the IPv6 default route."
  }
}

run "arm64_runtime_is_single_digest_bound_ssm_only_node" {
  command = plan

  module {
    source = "../modules/arm64-single-node"
  }

  variables {
    name                              = "avia-ipv6-fixture"
    region                            = "eu-central-1"
    subnet_id                         = "subnet-00000000000000001"
    security_group_id                 = "sg-00000000000000001"
    ami_id                            = "ami-0123456789abcdef0"
    ami_ssm_parameter_name            = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64"
    instance_type                     = "t4g.small"
    root_volume_size_gib              = 20
    delete_root_volume_on_termination = true
    ecr_repository_arns = [
      "arn:aws:ecr:eu-central-1:111122223333:repository/avia-ipv6-gateway",
      "arn:aws:ecr:eu-central-1:111122223333:repository/avia-ipv6-web-demo",
    ]
    image_uris = {
      gateway    = "111122223333.dkr-ecr.eu-central-1.on.aws/avia-ipv6-gateway@sha256:${format("%064d", 0)}"
      "web-demo" = "111122223333.dkr-ecr.eu-central-1.on.aws/avia-ipv6-web-demo@sha256:${format("%064d", 0)}"
    }
    cloudflared_image                  = "cloudflare/cloudflared@sha256:${format("%064d", 0)}"
    ecr_registry_host                  = "111122223333.dkr-ecr.eu-central-1.on.aws"
    cloudflare_connector_parameter_arn = "arn:aws:ssm:eu-central-1:111122223333:parameter/avia-ipv6-trial/cloudflare-connector"
    compose_bundle_path                = "/opt/aviasurveil360/aws-ipv6-trial/compose.yaml"
    compose_bundle_sha256              = format("%064d", 0)
    kms_key_arn                        = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
  }

  assert {
    condition     = aws_instance.runtime.instance_type == "t4g.small" && aws_instance.runtime.ipv6_address_count == 1 && !aws_instance.runtime.associate_public_ip_address
    error_message = "The runtime must be one IPv6-only t4g.small instance."
  }

  assert {
    condition     = aws_instance.runtime.metadata_options[0].http_tokens == "required" && aws_instance.runtime.metadata_options[0].http_protocol_ipv6 == "enabled"
    error_message = "The runtime must require IMDSv2 and enable the IMDS IPv6 endpoint."
  }

  assert {
    condition     = strcontains(base64decode(aws_instance.runtime.user_data), "TUNNEL_EDGE_IP_VERSION=6") && strcontains(base64decode(aws_instance.runtime.user_data), "@sha256:") && !strcontains(base64decode(aws_instance.runtime.user_data), "SECRET_VALUE")
    error_message = "User data must contain the IPv6 edge setting and image digests, never secret values."
  }
}

run "edge_runtime_auth_has_explicit_public_boundary" {
  command = plan

  module {
    source = "../modules/cloudflare-edge-runtime-auth"
  }

  variables {
    name                     = "avia-ipv6-fixture"
    cloudflare_account_id    = "cf-account-fixture"
    cloudflare_zone_id       = "cf-zone-fixture"
    hostname                 = "demo.aviacaa.example"
    connector_parameter_name = "/avia-ipv6-trial/cloudflare-connector"
    kms_key_arn              = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
    access_public_exposure   = true
    allowed_identities       = []
    allowed_domains          = []
  }

  assert {
    condition     = cloudflare_dns_record.hostname.proxied && cloudflare_dns_record.hostname.type == "CNAME"
    error_message = "The trial hostname must use a proxied CNAME to the tunnel."
  }

  assert {
    condition     = length(cloudflare_zero_trust_access_application.this) == 0 && length(cloudflare_zero_trust_access_policy.allow) == 0
    error_message = "Public exposure must be an explicit, empty-audience choice."
  }
}

run "budget_is_bounded_and_tag_filtered" {
  command = plan

  module {
    source = "../modules/trial-budget"
  }

  variables {
    name                  = "avia-ipv6-fixture"
    monthly_ceiling_usd   = 25
    one_run_ceiling_usd   = 10
    estimated_monthly_usd = 12
    estimated_one_run_usd = 5
    trial_expiry          = "2026-12-31T23:59:59Z"
    alert_recipients      = ["cost-owner@example.invalid"]
  }

  assert {
    condition     = aws_budgets_budget.trial.limit_amount == "25" && aws_budgets_budget.trial.cost_filter[0].values[0] == "user:TrialProfile$aws-ipv6-trial"
    error_message = "The budget must be bounded and scoped to the IPv6 trial tag."
  }
}
