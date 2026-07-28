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
    private_compute_subnet_ids = ["subnet-00000000000000011", "subnet-00000000000000012"]
  }
}

dependency "security" {
  config_path  = "../security"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    application_security_group_id = "sg-00000000000000002"
    instance_profile_name         = "avia-fixture-runtime"
  }
}

dependency "identity" {
  config_path  = "../identity-secrets"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    kms_key_arn = "arn:aws:kms:${include.root.locals.region}:${include.root.locals.account_id}:key/11111111-2222-3333-4444-555555555555"
    secret_arns = {
      application-runtime = "arn:aws:secretsmanager:${include.root.locals.region}:${include.root.locals.account_id}:secret:application-runtime"
      identity-runtime    = "arn:aws:secretsmanager:${include.root.locals.region}:${include.root.locals.account_id}:secret:identity-runtime"
    }
  }
}

dependency "load_balancer" {
  config_path  = "../load-balancer"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    target_group_arn = "arn:aws:elasticloadbalancing:${include.root.locals.region}:${include.root.locals.account_id}:targetgroup/avia-fixture/1111111111111111"
  }
}

dependency "service_endpoints" {
  config_path  = "../service-endpoints"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    interface_endpoint_ids = {
      ecr-api        = "vpce-00000000000000001"
      ecr-dkr        = "vpce-00000000000000002"
      logs           = "vpce-00000000000000003"
      secretsmanager = "vpce-00000000000000004"
      ssm            = "vpce-00000000000000005"
    }
    s3_endpoint_id = "vpce-00000000000000006"
  }
}

dependency "artifact" {
  config_path  = "../artifact-publication"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    image_uri   = include.root.locals.input_config.locals.image_uri
    sbom_sha256 = include.root.locals.input_config.locals.sbom_sha256
    scan_sha256 = include.root.locals.input_config.locals.scan_sha256
  }
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/compute"
}

inputs = {
  name                  = include.root.locals.input_config.locals.name_prefix
  region                = include.root.locals.region
  private_subnet_ids    = dependency.network.outputs.private_compute_subnet_ids
  security_group_ids    = [dependency.security.outputs.application_security_group_id]
  instance_profile_name = dependency.security.outputs.instance_profile_name
  instance_type         = include.root.locals.input_config.locals.instance_type
  ami_id                = include.root.locals.input_config.locals.ami_id
  image_uri             = dependency.artifact.outputs.image_uri
  secret_arns           = values(dependency.identity.outputs.secret_arns)
  kms_key_arn           = dependency.identity.outputs.kms_key_arn
  target_group_arns     = [dependency.load_balancer.outputs.target_group_arn]
  otel_endpoint         = include.root.locals.input_config.locals.otel_endpoint
  min_size              = include.root.locals.input_config.locals.min_size
  desired_capacity      = include.root.locals.input_config.locals.desired_capacity
  max_size              = include.root.locals.input_config.locals.max_size
  tags                  = include.root.locals.input_config.locals.tags
}

dependencies {
  paths = ["../service-endpoints"]
}
