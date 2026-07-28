variable "name_prefix" {
  description = "Stable name prefix for secret resources."
  type        = string
}

variable "secret_names" {
  description = "Names of secret containers; values are populated outside Terraform."
  type        = set(string)

  validation {
    condition     = length(var.secret_names) > 0 && alltrue([for name in var.secret_names : can(regex("^[a-z0-9][a-z0-9-]+$", name))])
    error_message = "secret_names must contain bounded lower-case names."
  }
}

variable "recovery_window_days" {
  description = "Secrets Manager recovery window."
  type        = number

  validation {
    condition     = var.recovery_window_days >= 7 && var.recovery_window_days <= 30
    error_message = "recovery_window_days must be between 7 and 30."
  }
}

variable "tags" {
  description = "Mandatory ownership and cost tags."
  type        = map(string)

  validation {
    condition = alltrue([
      for key in ["Environment", "Owner", "CostCenter", "DataClassification", "ManagedBy"] :
      contains(keys(var.tags), key) && trimspace(var.tags[key]) != ""
    ])
    error_message = "tags must include the mandatory ownership and cost keys."
  }
}

resource "aws_kms_key" "secrets" {
  description             = "${var.name_prefix} Secrets Manager encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = merge(var.tags, { Purpose = "identity-and-application-secrets" })
}

resource "aws_kms_alias" "secrets" {
  name          = "alias/${var.name_prefix}-secrets"
  target_key_id = aws_kms_key.secrets.key_id
}

resource "aws_secretsmanager_secret" "this" {
  for_each = var.secret_names

  name                    = "${var.name_prefix}/${each.value}"
  description             = "AviaSurveil360 ${each.value} reference; value is populated through an authorized operation"
  kms_key_id              = aws_kms_key.secrets.arn
  recovery_window_in_days = var.recovery_window_days

  tags = merge(var.tags, { SecretReference = each.value })
}

output "kms_key_arn" {
  description = "Secrets KMS key ARN."
  value       = aws_kms_key.secrets.arn
}

output "secret_arns" {
  description = "Secret ARNs keyed by reference name."
  value       = { for name, secret in aws_secretsmanager_secret.this : name => secret.arn }
}
