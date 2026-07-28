include "root" {
  path           = find_in_parent_folders("root.hcl")
  expose         = true
  merge_strategy = "deep"
}

dependency "database" {
  config_path  = "../database"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    database_arn = "arn:aws:rds:${include.root.locals.region}:${include.root.locals.account_id}:db:avia-fixture"
  }
}

dependency "object_storage" {
  config_path  = "../object-storage"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    application_bucket_arn = "arn:aws:s3:::avia-fixture-application"
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
  source = "${get_repo_root()}/infra/terraform/modules/backup"
}

inputs = {
  name_prefix           = include.root.locals.input_config.locals.name_prefix
  kms_key_arn           = dependency.identity.outputs.kms_key_arn
  resource_arns         = [dependency.database.outputs.database_arn, dependency.object_storage.outputs.application_bucket_arn]
  backup_retention_days = include.root.locals.input_config.locals.backup_retention_days
  tags                  = include.root.locals.input_config.locals.tags
}
