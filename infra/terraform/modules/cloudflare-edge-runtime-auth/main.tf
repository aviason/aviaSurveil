variable "name" {
  description = "Stable trial tunnel name."
  type        = string
}

variable "cloudflare_account_id" {
  description = "Exact Cloudflare account ID."
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Existing Cloudflare zone ID."
  type        = string
}

variable "hostname" {
  description = "Owner-approved public hostname."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9.-]+[a-z0-9]$", var.hostname)) && !strcontains(var.hostname, "example.invalid")
    error_message = "hostname must be a concrete owner-approved DNS name."
  }
}

variable "connector_parameter_name" {
  description = "SSM SecureString name receiving the sensitive Cloudflare connector token."
  type        = string

  validation {
    condition     = startswith(var.connector_parameter_name, "/") && !strcontains(var.connector_parameter_name, "secret-value")
    error_message = "connector_parameter_name must be a path-like SSM name, not a secret literal."
  }
}

variable "kms_key_arn" {
  description = "Customer-managed KMS key for the connector parameter."
  type        = string
}

variable "access_public_exposure" {
  description = "Explicit owner decision to expose the synthetic demo publicly."
  type        = bool
}

variable "allowed_identities" {
  description = "Exact email identities allowed by Cloudflare Access."
  type        = set(string)
  default     = []
}

variable "allowed_domains" {
  description = "Exact email domains allowed by Cloudflare Access."
  type        = set(string)
  default     = []
}

variable "tags" {
  description = "Mandatory ownership and cost tags."
  type        = map(string)

  validation {
    condition = alltrue([
      for key in ["Environment", "Owner", "CostCenter", "DataClassification", "ManagedBy"] :
      contains(keys(var.tags), key) && trimspace(var.tags[key]) != ""
    ])
    error_message = "tags must include non-empty Environment, Owner, CostCenter, DataClassification, and ManagedBy values."
  }
}

locals {
  access_include = concat(
    [for identity in sort(tolist(var.allowed_identities)) : { email = { email = identity } }],
    [for domain in sort(tolist(var.allowed_domains)) : { email_domain = { domain = domain } }],
  )
}

check "access_audience" {
  assert {
    condition     = var.access_public_exposure ? length(local.access_include) == 0 : length(local.access_include) > 0
    error_message = "access_public_exposure must be true only for an intentionally public trial, or false with at least one exact Access identity/domain."
  }
}

resource "cloudflare_zero_trust_tunnel_cloudflared" "this" {
  account_id = var.cloudflare_account_id
  name       = var.name
  config_src = "cloudflare"
}

data "cloudflare_zero_trust_tunnel_cloudflared_token" "this" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.this.id
}

resource "cloudflare_dns_record" "hostname" {
  zone_id = var.cloudflare_zone_id
  name    = var.hostname
  content = "${cloudflare_zero_trust_tunnel_cloudflared.this.id}.cfargotunnel.com"
  type    = "CNAME"
  ttl     = 1
  proxied = true
}

resource "cloudflare_zero_trust_tunnel_cloudflared_config" "this" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.this.id
  config = {
    ingress = [
      {
        hostname = var.hostname
        service  = "http://gateway:8080"
      },
      {
        service = "http_status:404"
      },
    ]
  }
}

resource "cloudflare_zero_trust_access_policy" "allow" {
  count      = var.access_public_exposure ? 0 : 1
  account_id = var.cloudflare_account_id
  name       = "${var.name}-allowlist"
  decision   = "allow"
  include    = local.access_include
}

resource "cloudflare_zero_trust_access_application" "this" {
  count            = var.access_public_exposure ? 0 : 1
  account_id       = var.cloudflare_account_id
  type             = "self_hosted"
  name             = "${var.name}-access"
  domain           = var.hostname
  session_duration = "24h"
  policies = [
    {
      id         = cloudflare_zero_trust_access_policy.allow[0].id
      precedence = 1
    },
  ]
}

resource "aws_ssm_parameter" "connector" {
  name        = var.connector_parameter_name
  description = "Sensitive Cloudflare Tunnel connector token for the IPv6 trial"
  type        = "SecureString"
  key_id      = var.kms_key_arn
  value       = data.cloudflare_zero_trust_tunnel_cloudflared_token.this.token
  tier        = "Standard"

  tags = merge(var.tags, {
    Name         = var.connector_parameter_name
    SecretClass  = "cloudflare-tunnel-connector"
    TrialProfile = "aws-ipv6-trial"
  })
}

output "tunnel_id" {
  description = "Cloudflare Tunnel identifier."
  value       = cloudflare_zero_trust_tunnel_cloudflared.this.id
}

output "hostname" {
  description = "Configured Cloudflare hostname."
  value       = cloudflare_dns_record.hostname.name
}

output "connector_parameter_arn" {
  description = "SSM parameter ARN; the sensitive value is never output."
  value       = aws_ssm_parameter.connector.arn
}

output "connector_parameter_name" {
  description = "SSM parameter name consumed by the EC2 instance role."
  value       = aws_ssm_parameter.connector.name
}
