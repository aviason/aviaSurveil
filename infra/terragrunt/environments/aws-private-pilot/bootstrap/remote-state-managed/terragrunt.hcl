locals {
  input_config = read_terragrunt_config(get_env("AVIA_AWS_PRIVATE_PILOT_BOOTSTRAP_MANAGED_INPUTS_FILE"))
  aws_profile  = local.input_config.locals.aws_profile
  account_id   = local.input_config.locals.account_id
  region       = local.input_config.locals.region
}

remote_state {
  backend      = "s3"
  disable_init = false

  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }

  config = {
    bucket              = local.input_config.locals.state_bucket_name
    key                 = "aws-private-pilot/bootstrap/remote-state/terraform.tfstate"
    region              = local.region
    profile             = local.aws_profile
    encrypt             = true
    kms_key_id          = local.input_config.locals.state_kms_key_arn
    use_lockfile        = true
    allowed_account_ids = [local.account_id]
  }
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<-EOF
    provider "aws" {
      profile             = "${local.aws_profile}"
      region              = "${local.region}"
      allowed_account_ids = ["${local.account_id}"]
    }
  EOF
}

terraform {
  source = "${get_repo_root()}/infra/terraform/bootstrap/remote-state"

  before_hook "deny_unreviewed_remote_action" {
    commands = ["plan", "apply", "destroy", "import"]
    execute  = ["bash", "${get_repo_root()}/scripts/check-aws-private-pilot-infrastructure.sh", "remote-hook"]
  }
}

inputs = {
  state_bucket_name = local.input_config.locals.state_bucket_name
  kms_alias         = local.input_config.locals.kms_alias
  tags              = local.input_config.locals.tags
}
