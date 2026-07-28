locals {
  account_config     = read_terragrunt_config("${get_repo_root()}/infra/terragrunt/environments/aws-trial/account.hcl")
  environment_config = read_terragrunt_config("${get_repo_root()}/infra/terragrunt/environments/aws-trial/environment.hcl")
  input_config       = read_terragrunt_config(get_env("AVIA_TG_INPUTS_FILE"))
  fixture_mode       = try(local.input_config.locals.fixture_mode, false)

  account_id = local.account_config.locals.account_id != "" ? local.account_config.locals.account_id : local.input_config.locals.account_id
  region     = local.input_config.locals.region

  state_bucket_name = local.input_config.locals.state_bucket_name
  state_kms_key_arn = local.input_config.locals.state_kms_key_arn
  plan_directory    = get_env("AVIA_TG_PLAN_DIR")
  plan_name         = replace(path_relative_to_include(), "/", "__")
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<-EOF
    provider "aws" {
      region = "${local.region}"

      skip_credentials_validation = ${local.fixture_mode}
      skip_metadata_api_check     = ${local.fixture_mode}
      skip_region_validation      = ${local.fixture_mode}
      skip_requesting_account_id  = ${local.fixture_mode}

      default_tags {
        tags = {
          Environment        = "${local.input_config.locals.environment}"
          Owner              = "${local.input_config.locals.owner}"
          CostCenter         = "${local.input_config.locals.cost_center}"
          DataClassification = "${local.input_config.locals.data_classification}"
          ManagedBy          = "terraform"
        }
      }
    }
  EOF
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
    bucket       = local.state_bucket_name
    key          = "aws-trial/${path_relative_to_include()}/terraform.tfstate"
    region       = local.region
    encrypt      = true
    kms_key_id   = local.state_kms_key_arn
    use_lockfile = true
  }
}

terraform {
  extra_arguments "plan_artifact" {
    commands  = ["plan"]
    arguments = ["-out=${local.plan_directory}/${local.plan_name}.tfplan"]
  }

  before_hook "preflight" {
    commands = ["plan"]
    execute  = ["bash", "${get_repo_root()}/scripts/check-terragrunt.sh", "hook-before"]
  }

  after_hook "policy" {
    commands     = ["plan"]
    execute      = ["bash", "${get_repo_root()}/scripts/check-terragrunt.sh", "hook-after", "${local.plan_directory}/${local.plan_name}.tfplan"]
    run_on_error = false
  }
}
