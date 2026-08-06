include "root" {
  path           = find_in_parent_folders("root.hcl")
  expose         = true
  merge_strategy = "deep"
}

generate "providers" {
  path      = "providers.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<-EOF
    provider "aws" {
      region                      = "${include.root.locals.region}"
      skip_credentials_validation = ${include.root.locals.fixture_mode}
      skip_metadata_api_check     = ${include.root.locals.fixture_mode}
      skip_region_validation      = ${include.root.locals.fixture_mode}
      skip_requesting_account_id  = ${include.root.locals.fixture_mode}
    }

    provider "cloudflare" {}
  EOF
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/cloudflare-edge-runtime-auth"
}

inputs = {
  name                     = "${include.root.locals.environment_config.locals.environment}"
  cloudflare_account_id    = include.root.locals.input_config.locals.cloudflare_account_id
  cloudflare_zone_id       = include.root.locals.input_config.locals.cloudflare_zone_id
  hostname                 = include.root.locals.input_config.locals.cloudflare_hostname
  connector_parameter_name = include.root.locals.input_config.locals.cloudflare_connector_parameter_name
  kms_key_arn              = include.root.locals.input_config.locals.runtime_kms_key_arn
  access_public_exposure   = include.root.locals.input_config.locals.cloudflare_access_public_exposure
  allowed_identities       = include.root.locals.input_config.locals.cloudflare_allowed_identities
  allowed_domains          = include.root.locals.input_config.locals.cloudflare_allowed_domains
  tags                     = include.root.locals.input_config.locals.tags
}
