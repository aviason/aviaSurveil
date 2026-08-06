include "root" {
  path           = find_in_parent_folders("root.hcl")
  expose         = true
  merge_strategy = "deep"
}

generate "aws_provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<-EOF
    provider "aws" {
      region                      = "${include.root.locals.region}"
      skip_credentials_validation = ${include.root.locals.fixture_mode}
      skip_metadata_api_check     = ${include.root.locals.fixture_mode}
      skip_region_validation      = ${include.root.locals.fixture_mode}
      skip_requesting_account_id  = ${include.root.locals.fixture_mode}
    }
  EOF
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/ipv6-trial-network"
}

inputs = {
  name                         = "${include.root.locals.environment_config.locals.environment}"
  vpc_ipv4_cidr                = include.root.locals.input_config.locals.vpc_ipv4_cidr
  availability_zone            = include.root.locals.input_config.locals.availability_zone
  cloudflare_tunnel_ipv6_cidrs = include.root.locals.input_config.locals.cloudflare_tunnel_ipv6_cidrs
  management_ipv6_cidrs        = include.root.locals.input_config.locals.management_ipv6_cidrs
  dns_ipv6_cidrs               = include.root.locals.input_config.locals.dns_ipv6_cidrs
  bootstrap_https_ipv6_cidrs   = include.root.locals.input_config.locals.bootstrap_https_ipv6_cidrs
  tags                         = include.root.locals.input_config.locals.tags
}
