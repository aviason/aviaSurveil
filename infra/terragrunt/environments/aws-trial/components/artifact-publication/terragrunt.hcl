include "root" {
  path           = find_in_parent_folders("root.hcl")
  expose         = true
  merge_strategy = "deep"
}

dependency "ecr" {
  config_path  = "../ecr"
  skip_outputs = include.root.locals.fixture_mode

  mock_outputs_allowed_terraform_commands = ["validate", "plan"]
  mock_outputs = {
    repository_urls = {
      runtime = "${include.root.locals.account_id}.dkr.ecr.${include.root.locals.region}.amazonaws.com/avia-fixture-runtime"
    }
  }
}

terraform {
  source = "${get_repo_root()}/infra/terraform/modules/artifact-contract"
}

inputs = {
  image_uri   = include.root.locals.input_config.locals.image_uri
  sbom_sha256 = include.root.locals.input_config.locals.sbom_sha256
  scan_sha256 = include.root.locals.input_config.locals.scan_sha256
}
