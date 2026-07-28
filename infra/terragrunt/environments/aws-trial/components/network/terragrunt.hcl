include "root" {
  path           = find_in_parent_folders("root.hcl")
  expose         = true
  merge_strategy = "deep"
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/network"
}

inputs = {
  name                  = include.root.locals.input_config.locals.name_prefix
  vpc_cidr              = include.root.locals.input_config.locals.vpc_cidr
  availability_zones    = include.root.locals.input_config.locals.availability_zones
  public_subnet_cidrs   = include.root.locals.input_config.locals.public_subnet_cidrs
  compute_subnet_cidrs  = include.root.locals.input_config.locals.compute_subnet_cidrs
  database_subnet_cidrs = include.root.locals.input_config.locals.database_subnet_cidrs
  enable_nat_gateway    = include.root.locals.input_config.locals.enable_nat_gateway
  single_nat_gateway    = include.root.locals.input_config.locals.single_nat_gateway
  tags                  = include.root.locals.input_config.locals.tags
}
