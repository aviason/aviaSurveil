locals {
  input_config   = read_terragrunt_config(get_env("AVIA_TG_INPUTS_FILE"))
  plan_directory = get_env("AVIA_TG_PLAN_DIR")
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<-EOF
    provider "aws" {
      region = "${local.input_config.locals.region}"

      skip_credentials_validation = ${local.input_config.locals.fixture_mode}
      skip_metadata_api_check     = ${local.input_config.locals.fixture_mode}
      skip_region_validation      = ${local.input_config.locals.fixture_mode}
      skip_requesting_account_id  = ${local.input_config.locals.fixture_mode}
    }
  EOF
}

terraform {
  source = "${get_repo_root()}/infra/terraform/bootstrap/remote-state"

  extra_arguments "plan_artifact" {
    commands  = ["plan"]
    arguments = ["-out=${local.plan_directory}/bootstrap__remote-state.tfplan"]
  }

  before_hook "preflight" {
    commands = ["plan"]
    execute  = ["bash", "${get_repo_root()}/scripts/check-terragrunt.sh", "hook-before"]
  }

  after_hook "policy" {
    commands     = ["plan"]
    execute      = ["bash", "${get_repo_root()}/scripts/check-terragrunt.sh", "hook-after", "${local.plan_directory}/bootstrap__remote-state.tfplan"]
    run_on_error = false
  }
}

inputs = {
  state_bucket_name = local.input_config.locals.state_bucket_name
  kms_alias         = "alias/${local.input_config.locals.name_prefix}-terraform-state"
  tags              = local.input_config.locals.tags
}
