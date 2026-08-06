locals {
  account_config     = read_terragrunt_config("${get_repo_root()}/infra/terragrunt/environments/aws-ipv6-trial/account.hcl")
  environment_config = read_terragrunt_config("${get_repo_root()}/infra/terragrunt/environments/aws-ipv6-trial/environment.hcl")
  input_config       = read_terragrunt_config(get_env("AVIA_IPV6_TG_INPUTS_FILE"))
  fixture_mode       = try(local.input_config.locals.fixture_mode, false)

  account_id      = local.account_config.locals.account_id != "" ? local.account_config.locals.account_id : local.input_config.locals.account_id
  region          = local.input_config.locals.region
  plan_directory  = get_env("AVIA_TG_PLAN_DIR")
  plan_name       = replace(path_relative_to_include(), "/", "__")
  state_bucket    = local.input_config.locals.state_bucket_name
  state_kms_key   = local.input_config.locals.state_kms_key_arn
  trial_namespace = "aws-ipv6-trial"
}

remote_state {
  backend      = local.fixture_mode ? "local" : "s3"
  disable_init = get_env("AVIA_TG_DISABLE_BACKEND", "false") == "true"

  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }

  config = local.fixture_mode ? {
    path = "${local.plan_directory}/${local.plan_name}.tfstate"
    } : {
    bucket       = local.state_bucket
    key          = "${local.trial_namespace}/${path_relative_to_include()}/terraform.tfstate"
    region       = local.region
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
    commands = ["plan"]
    execute  = ["bash", "${get_repo_root()}/scripts/check-aws-ipv6-trial-preflight.sh"]
  }

  before_hook "layout_preflight" {
    commands = ["plan"]
    execute  = ["bash", "${get_repo_root()}/scripts/check-aws-ipv6-trial-terragrunt.sh", "hook-before"]
  }

  after_hook "policy" {
    commands     = ["plan"]
    execute      = ["bash", "${get_repo_root()}/scripts/check-aws-ipv6-trial-terragrunt.sh", "hook-after", "${local.plan_directory}/${local.plan_name}.tfplan"]
    run_on_error = false
  }
}
