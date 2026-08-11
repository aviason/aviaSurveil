variable "name" {
  description = "Stable private-pilot resource prefix."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]{2,30}$", var.name))
    error_message = "name must be a bounded lower-case resource prefix."
  }
}

variable "kms_alias_prefix" {
  description = "Stable project KMS alias prefix allowed by the operator policy."
  type        = string

  validation {
    condition     = var.kms_alias_prefix == "aviasurveil360-private-pilot"
    error_message = "kms_alias_prefix is fixed to aviasurveil360-private-pilot."
  }
}

variable "aws_account_id" {
  description = "Exact owner-approved AWS account ID."
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.aws_account_id))
    error_message = "aws_account_id must contain exactly 12 digits."
  }
}

variable "region" {
  description = "Exact owner-approved AWS region."
  type        = string

  validation {
    condition     = can(regex("^[a-z]{2}(-gov)?-[a-z]+-[0-9]$", var.region))
    error_message = "region must be an explicit AWS region."
  }
}

variable "availability_zones" {
  description = "Workload AZ A followed by structural database AZ B."
  type        = list(string)

  validation {
    condition     = length(var.availability_zones) == 2 && length(distinct(var.availability_zones)) == 2
    error_message = "availability_zones must contain exactly two distinct zones."
  }
}

variable "vpc_cidr" {
  description = "Private IPv4 CIDR retained for RDS and the S3 Gateway Endpoint."
  type        = string

  validation {
    condition     = can(cidrnetmask(var.vpc_cidr)) && !strcontains(var.vpc_cidr, ":")
    error_message = "vpc_cidr must be an IPv4 CIDR."
  }
}

variable "dual_stack_app_subnet_cidr" {
  description = "Private IPv4 half of the sole dual-stack workload subnet in AZ A."
  type        = string

  validation {
    condition     = can(cidrnetmask(var.dual_stack_app_subnet_cidr)) && !strcontains(var.dual_stack_app_subnet_cidr, ":")
    error_message = "dual_stack_app_subnet_cidr must be an IPv4 CIDR."
  }
}

variable "private_database_subnet_cidrs" {
  description = "Exactly two private IPv4 database subnet CIDRs, ordered AZ A then B."
  type        = list(string)

  validation {
    condition     = length(var.private_database_subnet_cidrs) == 2 && alltrue([for cidr in var.private_database_subnet_cidrs : can(cidrnetmask(cidr)) && !strcontains(cidr, ":")])
    error_message = "private_database_subnet_cidrs must contain exactly two IPv4 CIDRs."
  }
}

variable "cloudflare_account_id" {
  description = "Exact owner-approved Cloudflare account ID."
  type        = string

  validation {
    condition     = can(regex("^[0-9a-f]{32}$", var.cloudflare_account_id))
    error_message = "cloudflare_account_id must be an explicit 32-character ID."
  }
}

variable "cloudflare_zone_id" {
  description = "Exact owner-approved Cloudflare zone ID."
  type        = string

  validation {
    condition     = can(regex("^[0-9a-f]{32}$", var.cloudflare_zone_id))
    error_message = "cloudflare_zone_id must be an explicit 32-character ID."
  }
}

variable "cloudflare_tunnel_name" {
  description = "Stable remotely managed Cloudflare Tunnel name."
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9][A-Za-z0-9._-]{2,99}$", var.cloudflare_tunnel_name))
    error_message = "cloudflare_tunnel_name must be an explicit bounded name."
  }
}

variable "cloudflare_dns_cutover_authorized" {
  description = "Explicit release-wave decision to manage the application DNS record. Keep false while an existing hostname route must remain untouched."
  type        = bool
}

variable "cloudflare_connector_parameter_name" {
  description = "SSM SecureString name used for the separately populated Cloudflare connector token."
  type        = string

  validation {
    condition     = startswith(var.cloudflare_connector_parameter_name, "/aviasurveil360/private-pilot/") && !strcontains(var.cloudflare_connector_parameter_name, "PENDING_SEPARATE_AUTHORIZATION")
    error_message = "cloudflare_connector_parameter_name must remain in the private-pilot namespace and must not contain a value."
  }
}

variable "cloudflare_tunnel_ipv6_cidrs" {
  description = "Fresh reviewed Cloudflare Tunnel edge IPv6 CIDRs allowed on TCP and UDP 7844."
  type        = set(string)

  validation {
    condition     = length(var.cloudflare_tunnel_ipv6_cidrs) > 0 && alltrue([for cidr in var.cloudflare_tunnel_ipv6_cidrs : can(cidrhost(cidr, 0)) && strcontains(cidr, ":") && cidr != "::/0"])
    error_message = "cloudflare_tunnel_ipv6_cidrs must contain reviewed non-default IPv6 CIDRs."
  }
}

variable "hostname" {
  description = "Owner-approved public hostname routed to Cloudflare Tunnel."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9.-]+[a-z0-9]$", var.hostname)) && !strcontains(var.hostname, "example.invalid")
    error_message = "hostname must be a concrete owner-approved DNS name."
  }
}

variable "ami_id" {
  description = "Freshly resolved, owner-approved Amazon Linux ARM64 AMI ID."
  type        = string

  validation {
    condition     = can(regex("^ami-[0-9a-f]{17}$", var.ami_id))
    error_message = "ami_id must be an explicit long-format AMI identifier."
  }
}

variable "instance_type" {
  description = "The one approved workload instance type."
  type        = string

  validation {
    condition     = var.instance_type == "t4g.small"
    error_message = "instance_type is fixed to t4g.small."
  }
}

variable "root_volume_size_gib" {
  description = "Bounded encrypted gp3 root volume size."
  type        = number

  validation {
    condition     = var.root_volume_size_gib == floor(var.root_volume_size_gib) && var.root_volume_size_gib >= 20 && var.root_volume_size_gib <= 64
    error_message = "root_volume_size_gib must be an integer between 20 and 64 GiB."
  }
}

variable "release_manifest_sha256" {
  description = "Digest binding the future offline-validated release manifest."
  type        = string

  validation {
    condition     = can(regex("^[0-9a-f]{64}$", var.release_manifest_sha256))
    error_message = "release_manifest_sha256 must be a lowercase SHA-256 digest."
  }
}

variable "database_engine_version" {
  description = "Owner-approved PostgreSQL engine version."
  type        = string

  validation {
    condition     = can(regex("^[0-9]+(\\.[0-9]+){0,2}$", var.database_engine_version))
    error_message = "database_engine_version must be explicit."
  }
}

variable "database_allocated_storage_gib" {
  description = "Initial encrypted gp3 storage for the one RDS instance."
  type        = number

  validation {
    condition     = var.database_allocated_storage_gib == floor(var.database_allocated_storage_gib) && var.database_allocated_storage_gib >= 20 && var.database_allocated_storage_gib <= 90
    error_message = "database_allocated_storage_gib must be an integer from 20 through 90 GiB so the 100 GiB autoscaling ceiling remains valid."
  }
}

variable "smtp_port" {
  description = "Owner-approved encrypted SMTP relay port used by application egress."
  type        = number

  validation {
    condition     = contains([465, 587], var.smtp_port)
    error_message = "smtp_port must be 465 or 587."
  }
}

variable "smtp_ipv6_cidrs" {
  description = "Exact owner-approved encrypted SMTP relay IPv6 destinations."
  type        = set(string)

  validation {
    condition     = length(var.smtp_ipv6_cidrs) > 0 && alltrue([for cidr in var.smtp_ipv6_cidrs : can(cidrhost(cidr, 0)) && strcontains(cidr, ":") && cidr != "::/0"])
    error_message = "smtp_ipv6_cidrs must contain reviewed non-default IPv6 CIDRs."
  }
}

variable "bucket_name_prefix" {
  description = "Globally unique owner-approved prefix for private pilot buckets."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]{8,45}[a-z0-9]$", var.bucket_name_prefix))
    error_message = "bucket_name_prefix must be a concrete S3-safe prefix."
  }
}

variable "budget_monthly_usd" {
  description = "Owner-approved monthly cost ceiling."
  type        = number

  validation {
    condition     = var.budget_monthly_usd > 0 && var.budget_monthly_usd <= 500
    error_message = "budget_monthly_usd must be positive and no more than 500 USD."
  }
}

variable "tags" {
  description = "Mandatory owner, cost, environment, and data-classification tags."
  type        = map(string)

  validation {
    condition = alltrue([
      for key in ["Environment", "Owner", "CostCenter", "DataClassification", "ManagedBy"] :
      contains(keys(var.tags), key) && trimspace(var.tags[key]) != ""
    ]) && try(var.tags["Environment"] == "production", false)
    error_message = "tags must contain all required keys and Environment=production."
  }
}

check "availability_zones_belong_to_region" {
  assert {
    condition     = alltrue([for zone in var.availability_zones : startswith(zone, var.region) && length(zone) == length(var.region) + 1 && can(regex("[a-z]$", zone))])
    error_message = "availability_zones must belong to the exact owner-approved region."
  }
}
