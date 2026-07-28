include "root" {
  path           = find_in_parent_folders("root.hcl")
  expose         = true
  merge_strategy = "deep"
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
  source = "${get_repo_root()}/infra/terraform/modules/observability"
}

inputs = {
  name_prefix        = include.root.locals.input_config.locals.name_prefix
  kms_key_arn        = dependency.identity.outputs.kms_key_arn
  log_retention_days = include.root.locals.input_config.locals.log_retention_days
  alarm_topic_arn    = include.root.locals.input_config.locals.alarm_topic_arn
  otel_endpoint      = include.root.locals.input_config.locals.otel_endpoint
  tags               = include.root.locals.input_config.locals.tags
}
