include "root" {
  path           = find_in_parent_folders("root.hcl")
  expose         = true
  merge_strategy = "deep"
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/identity-secrets"
}

inputs = {
  name_prefix          = include.root.locals.input_config.locals.name_prefix
  secret_names         = include.root.locals.input_config.locals.secret_names
  recovery_window_days = include.root.locals.input_config.locals.secret_recovery_days
  tags                 = include.root.locals.input_config.locals.tags
}
