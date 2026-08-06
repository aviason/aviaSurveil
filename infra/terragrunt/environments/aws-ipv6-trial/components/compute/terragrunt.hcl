include "root" {
  path           = find_in_parent_folders("root.hcl")
  expose         = true
  merge_strategy = "deep"
}

dependency "network" {
  config_path  = "../network"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    runtime_subnet_id         = "subnet-00000000000000001"
    runtime_security_group_id = "sg-00000000000000001"
  }
}

dependency "registry" {
  config_path  = "../registry"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    repository_arns = {
      gateway  = "arn:aws:ecr:fixture:111122223333:repository/aws-ipv6-trial-gateway"
      web_demo = "arn:aws:ecr:fixture:111122223333:repository/aws-ipv6-trial-web-demo"
    }
  }
}

dependency "edge_runtime_auth" {
  config_path  = "../edge-runtime-auth"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    connector_parameter_arn = "arn:aws:ssm:fixture:111122223333:parameter/aws-ipv6-trial/cloudflare-connector"
  }
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
  source = "${get_repo_root()}/infra/terraform/modules/arm64-single-node"
}

inputs = {
  name                               = "${include.root.locals.environment_config.locals.environment}"
  region                             = include.root.locals.region
  subnet_id                          = dependency.network.outputs.runtime_subnet_id
  security_group_id                  = dependency.network.outputs.runtime_security_group_id
  ami_ssm_parameter_name             = include.root.locals.input_config.locals.ami_ssm_parameter_name
  ami_id                             = include.root.locals.input_config.locals.ami_id
  instance_type                      = "t4g.small"
  root_volume_size_gib               = include.root.locals.input_config.locals.root_volume_size_gib
  delete_root_volume_on_termination  = include.root.locals.input_config.locals.delete_root_volume_on_termination
  ecr_repository_arns                = values(dependency.registry.outputs.repository_arns)
  image_uris                         = include.root.locals.input_config.locals.image_uris
  ecr_registry_host                  = include.root.locals.input_config.locals.ecr_registry_host
  cloudflared_image                  = include.root.locals.input_config.locals.cloudflared_image
  cloudflare_connector_parameter_arn = dependency.edge_runtime_auth.outputs.connector_parameter_arn
  compose_bundle_path                = include.root.locals.input_config.locals.compose_bundle_path
  compose_bundle_sha256              = include.root.locals.input_config.locals.compose_bundle_sha256
  kms_key_arn                        = include.root.locals.input_config.locals.runtime_kms_key_arn
  tags                               = include.root.locals.input_config.locals.tags
}

dependencies {
  paths = ["../network", "../registry", "../edge-runtime-auth"]
}
