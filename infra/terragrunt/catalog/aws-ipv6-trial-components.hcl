locals {
  aws_ipv6_trial_components = {
    network = {
      phase        = "foundation"
      dependencies = []
      source       = "infra/terraform/modules/ipv6-trial-network"
    }
    registry = {
      phase        = "foundation"
      dependencies = []
      source       = "infra/terraform/modules/ecr"
    }
    edge_runtime_auth = {
      phase        = "edge-runtime-auth"
      dependencies = []
      source       = "infra/terraform/modules/cloudflare-edge-runtime-auth"
    }
    budget = {
      phase        = "foundation"
      dependencies = []
      source       = "infra/terraform/modules/trial-budget"
    }
    compute = {
      phase = "compute"
      dependencies = [
        "network",
        "registry",
        "edge-runtime-auth",
      ]
      source = "infra/terraform/modules/arm64-single-node"
    }
  }
}
