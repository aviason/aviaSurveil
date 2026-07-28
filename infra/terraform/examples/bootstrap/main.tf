terraform {
  required_version = ">= 1.10, < 2.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 6.0, < 7.0"
    }
  }
}

provider "aws" {
  region = var.region

  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_region_validation      = true
  skip_requesting_account_id  = true
}

module "remote_state" {
  source = "../../bootstrap/remote-state"

  state_bucket_name = var.state_bucket_name
  kms_alias         = var.kms_alias
  tags              = var.tags
}

variable "region" {
  description = "Explicit approved AWS region."
  type        = string
}

variable "state_bucket_name" {
  description = "Globally unique remote-state bucket name."
  type        = string
}

variable "kms_alias" {
  description = "Approved remote-state KMS alias."
  type        = string
}

variable "tags" {
  description = "Mandatory ownership and cost tags."
  type        = map(string)
}

output "backend_contract" {
  description = "Values required by the generated Terragrunt backend."
  value       = module.remote_state.backend_contract
}
