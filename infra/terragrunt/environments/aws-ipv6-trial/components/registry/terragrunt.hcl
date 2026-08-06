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
  source = "${get_repo_root()}/infra/terraform/modules/ecr"
}

inputs = {
  name_prefix = "${include.root.locals.environment_config.locals.environment}"
  # Milestone 1 is intentionally limited to the gateway and web-demo images.
  repositories = toset(["gateway", "web-demo"])
  kms_key_arn  = include.root.locals.input_config.locals.ecr_kms_key_arn
  tags         = include.root.locals.input_config.locals.tags
}
