locals {
  input_config   = read_terragrunt_config(get_env("AVIA_AWS_PRIVATE_PILOT_BOOTSTRAP_INPUTS_FILE"))
  plan_directory = get_env("AVIA_TG_PLAN_DIR")
  aws_profile    = local.input_config.locals.aws_profile
  account_id     = local.input_config.locals.account_id
  region         = local.input_config.locals.region
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<-EOF
    provider "aws" {
      profile             = "${local.aws_profile}"
      region              = "${local.region}"
      allowed_account_ids = ["${local.account_id}"]

      default_tags {
        tags = {
          Environment  = "production"
          ManagedBy    = "terraform"
          PilotProfile = "aws-private-pilot"
        }
      }
    }
  EOF
}

terraform {
  source = "${get_repo_root()}/infra/terraform/bootstrap/remote-state"

  extra_arguments "saved_plan" {
    commands  = ["plan"]
    arguments = ["-input=false", "-out=${local.plan_directory}/bootstrap__remote-state.tfplan"]
  }

  before_hook "plan_authority" {
    commands = ["plan"]
    execute  = ["bash", "${get_repo_root()}/scripts/check-aws-private-pilot-infrastructure.sh", "bootstrap-plan-hook"]
  }

  before_hook "deny_mutation_without_reviewed_plan" {
    commands = ["apply", "destroy", "import"]
    execute  = ["bash", "${get_repo_root()}/scripts/check-aws-private-pilot-infrastructure.sh", "remote-hook"]
  }
}

inputs = {
  state_bucket_name = local.input_config.locals.state_bucket_name
  kms_alias         = "alias/${local.input_config.locals.name_prefix}-terraform-state"
  tags              = local.input_config.locals.tags
}
