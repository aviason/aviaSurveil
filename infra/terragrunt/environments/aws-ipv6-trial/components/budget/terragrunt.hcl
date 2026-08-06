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
  source = "${get_repo_root()}/infra/terraform/modules/trial-budget"
}

inputs = {
  name                  = "${include.root.locals.environment_config.locals.environment}"
  monthly_ceiling_usd   = include.root.locals.input_config.locals.monthly_ceiling_usd
  one_run_ceiling_usd   = include.root.locals.input_config.locals.one_run_ceiling_usd
  estimated_monthly_usd = include.root.locals.input_config.locals.estimated_monthly_usd
  estimated_one_run_usd = include.root.locals.input_config.locals.estimated_one_run_usd
  trial_expiry          = include.root.locals.input_config.locals.trial_expiry
  alert_recipients      = include.root.locals.input_config.locals.alert_recipients
  tags                  = include.root.locals.input_config.locals.tags
}
