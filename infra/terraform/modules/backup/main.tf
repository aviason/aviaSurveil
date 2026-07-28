variable "name_prefix" {
  description = "Stable name prefix for backup resources."
  type        = string
}

variable "kms_key_arn" {
  description = "KMS key for the backup vault."
  type        = string
}

variable "resource_arns" {
  description = "Exact resource ARNs included in the backup selection."
  type        = list(string)

  validation {
    condition     = length(var.resource_arns) > 0 && alltrue([for arn in var.resource_arns : startswith(arn, "arn:")])
    error_message = "resource_arns must contain explicit ARNs."
  }
}

variable "backup_retention_days" {
  description = "Approved backup retention."
  type        = number

  validation {
    condition     = var.backup_retention_days >= 30
    error_message = "backup_retention_days must be at least 30."
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

resource "aws_backup_vault" "this" {
  name        = "${var.name_prefix}-vault"
  kms_key_arn = var.kms_key_arn

  tags = var.tags
}

resource "aws_backup_vault_lock_configuration" "this" {
  backup_vault_name   = aws_backup_vault.this.name
  min_retention_days  = var.backup_retention_days
  max_retention_days  = var.backup_retention_days * 4
  changeable_for_days = 7
}

resource "aws_backup_plan" "this" {
  name = "${var.name_prefix}-plan"

  rule {
    rule_name         = "daily"
    target_vault_name = aws_backup_vault.this.name
    schedule          = "cron(0 2 * * ? *)"
    start_window      = 60
    completion_window = 360

    lifecycle {
      delete_after = var.backup_retention_days
    }

    recovery_point_tags = var.tags
  }

  tags = var.tags
}

resource "aws_iam_role" "backup" {
  name_prefix = "${var.name_prefix}-backup-"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Service = "backup.amazonaws.com"
      }
      Action = "sts:AssumeRole"
    }]
  })

  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "backup" {
  role       = aws_iam_role.backup.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSBackupServiceRolePolicyForBackup"
}

resource "aws_backup_selection" "this" {
  name         = "${var.name_prefix}-resources"
  iam_role_arn = aws_iam_role.backup.arn
  plan_id      = aws_backup_plan.this.id
  resources    = var.resource_arns
}

output "vault_arn" {
  description = "Encrypted backup vault ARN."
  value       = aws_backup_vault.this.arn
}

output "plan_id" {
  description = "Backup plan identifier."
  value       = aws_backup_plan.this.id
}
