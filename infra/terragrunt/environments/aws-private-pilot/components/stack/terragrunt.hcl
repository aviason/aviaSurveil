include "root" {
  path           = find_in_parent_folders("root.hcl")
  expose         = true
  merge_strategy = "deep"
}

generate "providers" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<-EOF
    provider "aws" {
      profile                     = "${include.root.locals.aws_profile}"
      region                      = "${include.root.locals.region}"
      allowed_account_ids         = ["${include.root.locals.account_id}"]
      skip_credentials_validation = ${include.root.locals.fixture_mode}
      skip_metadata_api_check     = ${include.root.locals.fixture_mode}
      skip_region_validation      = ${include.root.locals.fixture_mode}
      skip_requesting_account_id  = ${include.root.locals.fixture_mode}

      default_tags {
        tags = {
          Environment = "production"
          ManagedBy   = "terraform"
          PilotProfile = "aws-private-pilot"
        }
      }
    }

    provider "cloudflare" {}
  EOF
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/aws-private-pilot"
}

inputs = {
  name                                = include.root.locals.input_config.locals.name
  kms_alias_prefix                    = include.root.locals.input_config.locals.kms_alias_prefix
  aws_account_id                      = include.root.locals.account_id
  region                              = include.root.locals.region
  availability_zones                  = include.root.locals.input_config.locals.availability_zones
  vpc_cidr                            = include.root.locals.input_config.locals.vpc_cidr
  dual_stack_app_subnet_cidr          = include.root.locals.input_config.locals.dual_stack_app_subnet_cidr
  private_database_subnet_cidrs       = include.root.locals.input_config.locals.private_database_subnet_cidrs
  cloudflare_account_id               = include.root.locals.input_config.locals.cloudflare_account_id
  cloudflare_zone_id                  = include.root.locals.input_config.locals.cloudflare_zone_id
  cloudflare_tunnel_name              = include.root.locals.input_config.locals.cloudflare_tunnel_name
  cloudflare_dns_cutover_authorized   = include.root.locals.input_config.locals.cloudflare_dns_cutover_authorized
  cloudflare_connector_parameter_name = include.root.locals.input_config.locals.cloudflare_connector_parameter_name
  cloudflare_tunnel_ipv6_cidrs        = include.root.locals.input_config.locals.cloudflare_tunnel_ipv6_cidrs
  hostname                            = include.root.locals.input_config.locals.hostname
  ami_id                              = include.root.locals.input_config.locals.ami_id
  instance_type                       = include.root.locals.input_config.locals.instance_type
  root_volume_size_gib                = include.root.locals.input_config.locals.root_volume_size_gib
  release_manifest_sha256             = include.root.locals.input_config.locals.release_manifest_sha256
  database_engine_version             = include.root.locals.input_config.locals.database_engine_version
  database_allocated_storage_gib      = include.root.locals.input_config.locals.database_allocated_storage_gib
  smtp_port                           = include.root.locals.input_config.locals.smtp_port
  smtp_ipv6_cidrs                     = include.root.locals.input_config.locals.smtp_ipv6_cidrs
  bucket_name_prefix                  = include.root.locals.input_config.locals.bucket_name_prefix
  budget_monthly_usd                  = include.root.locals.input_config.locals.budget_monthly_usd
  tags                                = include.root.locals.input_config.locals.tags
}
