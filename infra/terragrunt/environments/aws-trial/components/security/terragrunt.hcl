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
    vpc_id = "vpc-0123456789abcdef0"
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

dependency "ecr" {
  config_path  = "../ecr"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    repository_arns = {
      runtime = "arn:aws:ecr:${include.root.locals.region}:${include.root.locals.account_id}:repository/avia-fixture-runtime"
    }
  }
}

dependency "object_storage" {
  config_path  = "../object-storage"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    application_bucket_arn = "arn:aws:s3:::avia-fixture-application"
    backup_bucket_arn      = "arn:aws:s3:::avia-fixture-backup"
  }
}

dependency "observability" {
  config_path  = "../observability"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    log_group_arns = {
      application = "arn:aws:logs:${include.root.locals.region}:${include.root.locals.account_id}:log-group:/avia/fixture/application:*"
      gateway     = "arn:aws:logs:${include.root.locals.region}:${include.root.locals.account_id}:log-group:/avia/fixture/gateway:*"
      keycloak    = "arn:aws:logs:${include.root.locals.region}:${include.root.locals.account_id}:log-group:/avia/fixture/keycloak:*"
      worker      = "arn:aws:logs:${include.root.locals.region}:${include.root.locals.account_id}:log-group:/avia/fixture/worker:*"
    }
  }
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/security"
}

inputs = {
  name                = include.root.locals.input_config.locals.name_prefix
  vpc_id              = dependency.network.outputs.vpc_id
  application_port    = include.root.locals.input_config.locals.application_port
  database_port       = 5432
  secret_arns         = values(dependency.identity.outputs.secret_arns)
  bucket_arns         = [dependency.object_storage.outputs.application_bucket_arn, dependency.object_storage.outputs.backup_bucket_arn]
  kms_key_arns        = [dependency.identity.outputs.kms_key_arn]
  ecr_repository_arns = values(dependency.ecr.outputs.repository_arns)
  log_group_arns      = values(dependency.observability.outputs.log_group_arns)
  tags                = include.root.locals.input_config.locals.tags
}
