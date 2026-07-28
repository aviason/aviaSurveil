variable "name_prefix" {
  description = "Stable prefix for repository names."
  type        = string
}

variable "repositories" {
  description = "Approved runtime repository suffixes."
  type        = set(string)

  validation {
    condition     = length(var.repositories) > 0
    error_message = "At least one ECR repository is required."
  }
}

variable "kms_key_arn" {
  description = "KMS key used for ECR encryption."
  type        = string
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

resource "aws_ecr_repository" "this" {
  for_each = var.repositories

  name                 = "${var.name_prefix}-${each.value}"
  image_tag_mutability = "IMMUTABLE"

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = var.kms_key_arn
  }

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = merge(var.tags, { Component = each.value })
}

resource "aws_ecr_lifecycle_policy" "this" {
  for_each = aws_ecr_repository.this

  repository = each.value.name
  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Expire untagged trial images after seven days"
        selection = {
          tagStatus   = "untagged"
          countType   = "sinceImagePushed"
          countUnit   = "days"
          countNumber = 7
        }
        action = {
          type = "expire"
        }
      },
    ]
  })
}

output "repository_arns" {
  description = "Repository ARNs keyed by component."
  value       = { for key, repository in aws_ecr_repository.this : key => repository.arn }
}

output "repository_urls" {
  description = "Repository URLs keyed by component."
  value       = { for key, repository in aws_ecr_repository.this : key => repository.repository_url }
}
