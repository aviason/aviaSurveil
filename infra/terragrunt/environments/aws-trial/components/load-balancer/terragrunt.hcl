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
    vpc_id            = "vpc-0123456789abcdef0"
    public_subnet_ids = ["subnet-00000000000000001", "subnet-00000000000000002"]
  }
}

dependency "security" {
  config_path  = "../security"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    alb_security_group_id = "sg-00000000000000001"
  }
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/load-balancer"
}

inputs = {
  name              = include.root.locals.input_config.locals.name_prefix
  vpc_id            = dependency.network.outputs.vpc_id
  public_subnet_ids = dependency.network.outputs.public_subnet_ids
  security_group_id = dependency.security.outputs.alb_security_group_id
  certificate_arn   = include.root.locals.input_config.locals.certificate_arn
  target_port       = include.root.locals.input_config.locals.application_port
  tags              = include.root.locals.input_config.locals.tags
}
