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
    vpc_id                          = "vpc-0123456789abcdef0"
    private_compute_subnet_ids      = ["subnet-00000000000000011", "subnet-00000000000000012"]
    private_compute_route_table_ids = ["rtb-00000000000000011", "rtb-00000000000000012"]
  }
}

dependency "security" {
  config_path  = "../security"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    application_security_group_id = "sg-00000000000000002"
  }
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/service-endpoints"
}

inputs = {
  name                          = include.root.locals.input_config.locals.name_prefix
  region                        = include.root.locals.region
  vpc_id                        = dependency.network.outputs.vpc_id
  private_subnet_ids            = dependency.network.outputs.private_compute_subnet_ids
  private_route_table_ids       = dependency.network.outputs.private_compute_route_table_ids
  application_security_group_id = dependency.security.outputs.application_security_group_id
  tags                          = include.root.locals.input_config.locals.tags
}
