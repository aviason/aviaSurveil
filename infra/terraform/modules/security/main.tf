variable "name" {
  description = "Stable name prefix for security resources."
  type        = string
}

variable "vpc_id" {
  description = "VPC containing the security groups."
  type        = string
}

variable "application_port" {
  description = "Private application listener port."
  type        = number
}

variable "database_port" {
  description = "Private PostgreSQL listener port."
  type        = number
}

variable "secret_arns" {
  description = "Exact Secrets Manager resources available to the runtime."
  type        = list(string)
}

variable "bucket_arns" {
  description = "Exact S3 bucket resources available to the runtime."
  type        = list(string)
}

variable "kms_key_arns" {
  description = "Exact KMS resources available to the runtime."
  type        = list(string)
}

variable "ecr_repository_arns" {
  description = "Exact ECR repositories available to the runtime."
  type        = list(string)
}

variable "log_group_arns" {
  description = "Exact CloudWatch log resources available to the runtime."
  type        = list(string)
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

resource "aws_security_group" "alb" {
  name_prefix = "${var.name}-alb-"
  description = "HTTPS ingress to the AviaSurveil360 trial ALB"
  vpc_id      = var.vpc_id

  tags = merge(var.tags, { Name = "${var.name}-alb" })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "application" {
  name_prefix = "${var.name}-application-"
  description = "Private application instances"
  vpc_id      = var.vpc_id

  tags = merge(var.tags, { Name = "${var.name}-application" })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "database" {
  name_prefix = "${var.name}-database-"
  description = "Private PostgreSQL access from application instances"
  vpc_id      = var.vpc_id

  tags = merge(var.tags, { Name = "${var.name}-database" })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_ingress_rule" "alb_https" {
  security_group_id = aws_security_group.alb.id
  description       = "Approved public HTTPS ingress"
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = 443
  ip_protocol       = "tcp"
  to_port           = 443
}

resource "aws_vpc_security_group_egress_rule" "alb_application" {
  security_group_id            = aws_security_group.alb.id
  description                  = "ALB to private application target"
  referenced_security_group_id = aws_security_group.application.id
  from_port                    = var.application_port
  ip_protocol                  = "tcp"
  to_port                      = var.application_port
}

resource "aws_vpc_security_group_ingress_rule" "application" {
  for_each = { alb = aws_security_group.alb.id }

  security_group_id            = aws_security_group.application.id
  description                  = "Application traffic from the ALB"
  referenced_security_group_id = each.value
  from_port                    = var.application_port
  ip_protocol                  = "tcp"
  to_port                      = var.application_port
}

resource "aws_vpc_security_group_egress_rule" "application_database" {
  security_group_id            = aws_security_group.application.id
  description                  = "Application to PostgreSQL"
  referenced_security_group_id = aws_security_group.database.id
  from_port                    = var.database_port
  ip_protocol                  = "tcp"
  to_port                      = var.database_port
}

resource "aws_vpc_security_group_ingress_rule" "database" {
  for_each = { application = aws_security_group.application.id }

  security_group_id            = aws_security_group.database.id
  description                  = "PostgreSQL from application instances"
  referenced_security_group_id = each.value
  from_port                    = var.database_port
  ip_protocol                  = "tcp"
  to_port                      = var.database_port
}

resource "aws_iam_role" "runtime" {
  name_prefix = "${var.name}-runtime-"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
      Action = "sts:AssumeRole"
    }]
  })

  tags = var.tags
}

locals {
  bucket_object_arns = [for arn in var.bucket_arns : "${arn}/*"]
}

resource "aws_iam_policy" "runtime" {
  name_prefix = "${var.name}-runtime-"
  description = "Resource-scoped trial runtime access"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ReadSecretReferences"
        Effect   = "Allow"
        Action   = ["secretsmanager:DescribeSecret", "secretsmanager:GetSecretValue"]
        Resource = var.secret_arns
      },
      {
        Sid      = "UseApprovedKmsKeys"
        Effect   = "Allow"
        Action   = ["kms:Decrypt", "kms:DescribeKey", "kms:GenerateDataKey"]
        Resource = var.kms_key_arns
      },
      {
        Sid      = "UseApprovedBuckets"
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject", "s3:ListBucket"]
        Resource = concat(var.bucket_arns, local.bucket_object_arns)
      },
      {
        Sid      = "PullApprovedImages"
        Effect   = "Allow"
        Action   = ["ecr:BatchCheckLayerAvailability", "ecr:BatchGetImage", "ecr:GetDownloadUrlForLayer"]
        Resource = var.ecr_repository_arns
      },
      {
        Sid      = "WriteApprovedLogs"
        Effect   = "Allow"
        Action   = ["logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = var.log_group_arns
      },
    ]
  })

  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "runtime" {
  role       = aws_iam_role.runtime.name
  policy_arn = aws_iam_policy.runtime.arn
}

resource "aws_iam_instance_profile" "runtime" {
  name_prefix = "${var.name}-runtime-"
  role        = aws_iam_role.runtime.name

  tags = var.tags
}

output "alb_security_group_id" {
  description = "ALB security group identifier."
  value       = aws_security_group.alb.id
}

output "application_security_group_id" {
  description = "Application security group identifier."
  value       = aws_security_group.application.id
}

output "database_security_group_id" {
  description = "Database security group identifier."
  value       = aws_security_group.database.id
}

output "runtime_role_arn" {
  description = "Resource-scoped runtime role ARN."
  value       = aws_iam_role.runtime.arn
}

output "instance_profile_name" {
  description = "Runtime instance profile name."
  value       = aws_iam_instance_profile.runtime.name
}
