variable "name" {
  description = "Stable database identifier."
  type        = string
}

variable "subnet_ids" {
  description = "Isolated database subnet identifiers."
  type        = list(string)

  validation {
    condition     = length(var.subnet_ids) >= 2
    error_message = "subnet_ids must span at least two subnets."
  }
}

variable "security_group_ids" {
  description = "Database security group identifiers."
  type        = list(string)
}

variable "kms_key_arn" {
  description = "KMS key for storage, managed credentials, and performance insights."
  type        = string
}

variable "instance_class" {
  description = "Approved RDS instance class."
  type        = string
}

variable "engine_version" {
  description = "Approved PostgreSQL engine version."
  type        = string
}

variable "allocated_storage" {
  description = "Initial encrypted storage in GiB."
  type        = number
}

variable "backup_retention_days" {
  description = "Automated backup retention in days."
  type        = number

  validation {
    condition     = var.backup_retention_days >= 7
    error_message = "backup_retention_days must be at least seven."
  }
}

variable "deletion_protection" {
  description = "Required database deletion protection."
  type        = bool

  validation {
    condition     = var.deletion_protection
    error_message = "deletion_protection must remain enabled."
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

resource "aws_db_subnet_group" "this" {
  name_prefix = "${var.name}-"
  description = "Isolated AviaSurveil360 database subnets"
  subnet_ids  = var.subnet_ids

  tags = var.tags
}

resource "aws_db_instance" "this" {
  identifier_prefix = "${var.name}-"

  engine         = "postgres"
  engine_version = var.engine_version
  instance_class = var.instance_class

  db_name  = "aviasurveil360"
  username = "aviasurveil360"
  port     = 5432

  manage_master_user_password   = true
  master_user_secret_kms_key_id = var.kms_key_arn

  allocated_storage     = var.allocated_storage
  max_allocated_storage = max(var.allocated_storage * 2, 100)
  storage_type          = "gp3"
  storage_encrypted     = true
  kms_key_id            = var.kms_key_arn

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = var.security_group_ids
  publicly_accessible    = false
  multi_az               = true

  backup_retention_period = var.backup_retention_days
  backup_window           = "01:00-02:00"
  maintenance_window      = "sun:03:00-sun:04:00"

  deletion_protection       = var.deletion_protection
  skip_final_snapshot       = false
  final_snapshot_identifier = "${var.name}-final"
  copy_tags_to_snapshot     = true

  enabled_cloudwatch_logs_exports       = ["postgresql", "upgrade"]
  performance_insights_enabled          = true
  performance_insights_kms_key_id       = var.kms_key_arn
  performance_insights_retention_period = 7

  auto_minor_version_upgrade = true
  apply_immediately          = false

  tags = var.tags
}

output "database_arn" {
  description = "RDS database ARN."
  value       = aws_db_instance.this.arn
}

output "endpoint" {
  description = "Private PostgreSQL endpoint."
  value       = aws_db_instance.this.endpoint
}

output "master_secret_arn" {
  description = "AWS-managed master credential secret ARN."
  value       = try(aws_db_instance.this.master_user_secret[0].secret_arn, null)
}
