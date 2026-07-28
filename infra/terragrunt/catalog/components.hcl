locals {
  components = {
    remote_state = {
      phase        = "bootstrap"
      dependencies = []
      source       = "infra/terraform/bootstrap/remote-state"
    }
    network = {
      phase        = "foundation"
      dependencies = []
      source       = "infra/terraform/modules/network"
    }
    identity_secrets = {
      phase        = "foundation"
      dependencies = []
      source       = "infra/terraform/modules/identity-secrets"
    }
    ecr = {
      phase        = "foundation"
      dependencies = ["identity-secrets"]
      source       = "infra/terraform/modules/ecr"
    }
    object_storage = {
      phase        = "foundation"
      dependencies = ["identity-secrets"]
      source       = "infra/terraform/modules/object-storage"
    }
    observability = {
      phase        = "foundation"
      dependencies = ["identity-secrets"]
      source       = "infra/terraform/modules/observability"
    }
    security = {
      phase        = "foundation"
      dependencies = ["network", "identity-secrets", "ecr", "object-storage", "observability"]
      source       = "infra/terraform/modules/security"
    }
    service_endpoints = {
      phase        = "foundation"
      dependencies = ["network", "security"]
      source       = "infra/terraform/modules/service-endpoints"
    }
    load_balancer = {
      phase        = "foundation"
      dependencies = ["network", "security"]
      source       = "infra/terraform/modules/load-balancer"
    }
    artifact_publication = {
      phase        = "artifact-publication"
      dependencies = ["ecr"]
      source       = "infra/terraform/modules/artifact-contract"
    }
    database = {
      phase        = "data-runtime"
      dependencies = ["network", "security", "identity-secrets"]
      source       = "infra/terraform/modules/database"
    }
    compute = {
      phase = "data-runtime"
      dependencies = [
        "network",
        "security",
        "service-endpoints",
        "load-balancer",
        "identity-secrets",
        "artifact-publication",
      ]
      source = "infra/terraform/modules/compute"
    }
    backup = {
      phase        = "data-runtime"
      dependencies = ["database", "object-storage", "identity-secrets"]
      source       = "infra/terraform/modules/backup"
    }
  }
}
