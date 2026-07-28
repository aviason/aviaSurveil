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
    private_database_subnet_ids = ["subnet-00000000000000021", "subnet-00000000000000022"]
  }
}

dependency "security" {
  config_path  = "../security"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    database_security_group_id = "sg-00000000000000003"
  }
}

dependency "identity" {
  config_path  = "../identity-secrets"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    kms_key_arn = "arn:aws:kms:${include.root.locals.region}:${include.root.locals.account_id}:key/11111111-2222-3333-4444-555555555555"
  }
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/database"
}

inputs = {
  name                  = include.root.locals.input_config.locals.name_prefix
  subnet_ids            = dependency.network.outputs.private_database_subnet_ids
  security_group_ids    = [dependency.security.outputs.database_security_group_id]
  kms_key_arn           = dependency.identity.outputs.kms_key_arn
  instance_class        = include.root.locals.input_config.locals.database_instance_class
  engine_version        = include.root.locals.input_config.locals.database_engine_version
  allocated_storage     = include.root.locals.input_config.locals.database_storage_gib
  backup_retention_days = include.root.locals.input_config.locals.backup_retention_days
  deletion_protection   = true
  tags                  = include.root.locals.input_config.locals.tags
}
