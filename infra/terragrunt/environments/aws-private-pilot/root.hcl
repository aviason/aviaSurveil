locals {
  account_config     = read_terragrunt_config("${get_repo_root()}/infra/terragrunt/environments/aws-private-pilot/account.hcl")
  environment_config = read_terragrunt_config("${get_repo_root()}/infra/terragrunt/environments/aws-private-pilot/environment.hcl")
  input_config       = read_terragrunt_config(get_env("AVIA_AWS_PRIVATE_PILOT_TG_INPUTS_FILE"))
  fixture_mode       = try(local.input_config.locals.fixture_mode, false)

  aws_profile    = "avia"
  account_id     = local.account_config.locals.account_id != "" ? local.account_config.locals.account_id : local.input_config.locals.account_id
  region         = local.input_config.locals.region
  plan_directory = get_env("AVIA_TG_PLAN_DIR")
  plan_name      = replace(path_relative_to_include(), "/", "__")
  state_bucket   = local.input_config.locals.state_bucket_name
  state_kms_key  = local.input_config.locals.state_kms_key_arn
}

remote_state {
  backend      = local.fixture_mode ? "local" : "s3"
  disable_init = get_env("AVIA_TG_DISABLE_BACKEND", "true") == "true"

  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }

  config = local.fixture_mode ? {
    path = "${local.plan_directory}/${local.plan_name}.tfstate"
    } : {
    bucket       = local.state_bucket
    key          = "aws-private-pilot/${path_relative_to_include()}/terraform.tfstate"
    region       = local.region
    profile      = local.aws_profile
    encrypt      = true
    kms_key_id   = local.state_kms_key
    use_lockfile = true
  }
}

terraform {
  extra_arguments "plan_artifact" {
    commands  = ["plan"]
    arguments = ["-out=${local.plan_directory}/${local.plan_name}.tfplan"]
  }

  before_hook "decision_preflight" {
    commands = ["plan", "apply", "destroy"]
    execute  = ["bash", "${get_repo_root()}/scripts/check-aws-private-pilot-decisions.sh"]
  }

  before_hook "remote_authority" {
    commands = ["plan", "apply", "destroy"]
    execute  = ["bash", "${get_repo_root()}/scripts/check-aws-private-pilot-infrastructure.sh", "remote-hook"]
  }
}
