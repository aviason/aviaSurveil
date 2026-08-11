locals {
  object_buckets = toset([
    "quarantine",
    "canonical",
    "attachments",
    "documents",
  ])
  runtime_secret_names = toset([
    "app-database-password",
    "app-migration-password",
    "keycloak-database-password",
    "oidc-client-secret",
    "session-encryption-key",
    "keycloak-service-client-secret",
    "app-smtp-password",
    "keycloak-smtp-password",
  ])
  runtime_repositories = toset([
    "cloudflared",
    "gateway",
    "application",
    "keycloak",
    "database-bootstrap",
  ])
}

resource "aws_kms_key" "data" {
  description             = "${var.name} private-pilot data, database, EBS, and backup encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = merge(var.tags, { Purpose = "private-pilot-data" })

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_kms_alias" "data" {
  name          = "alias/${var.kms_alias_prefix}-data"
  target_key_id = aws_kms_key.data.key_id
}

resource "aws_kms_key" "secrets" {
  description             = "${var.name} private-pilot runtime secret encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = merge(var.tags, { Purpose = "private-pilot-secrets" })

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_kms_alias" "secrets" {
  name          = "alias/${var.kms_alias_prefix}-secrets"
  target_key_id = aws_kms_key.secrets.key_id
}

resource "aws_kms_key" "logs" {
  description             = "${var.name} private-pilot CloudWatch log encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AccountKeyAdministration"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::${var.aws_account_id}:root" }
        Action    = "kms:*"
        Resource  = "*"
      },
      {
        Sid       = "CloudWatchLogsEncryption"
        Effect    = "Allow"
        Principal = { Service = "logs.${var.region}.amazonaws.com" }
        Action = [
          "kms:Decrypt",
          "kms:DescribeKey",
          "kms:Encrypt",
          "kms:GenerateDataKey",
          "kms:ReEncryptFrom",
          "kms:ReEncryptTo",
        ]
        Resource = "*"
        Condition = {
          ArnLike = {
            "kms:EncryptionContext:aws:logs:arn" = "arn:aws:logs:${var.region}:${var.aws_account_id}:log-group:/aviasurveil360/${var.name}/*"
          }
        }
      },
    ]
  })

  tags = merge(var.tags, { Purpose = "private-pilot-logs" })

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_kms_alias" "logs" {
  name          = "alias/${var.kms_alias_prefix}-logs"
  target_key_id = aws_kms_key.logs.key_id
}

resource "aws_secretsmanager_secret" "runtime" {
  for_each = local.runtime_secret_names

  name                    = "${var.name}/${each.value}"
  description             = "Empty private-pilot secret container; value population requires separate authorization"
  kms_key_id              = aws_kms_key.secrets.arn
  recovery_window_in_days = 30

  tags = merge(var.tags, { SecretReference = each.value })

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_s3_bucket" "objects" {
  for_each = local.object_buckets

  bucket        = "${var.bucket_name_prefix}-${each.value}"
  force_destroy = false

  tags = merge(var.tags, {
    Purpose = "private-pilot-${each.value}"
    Records = "append-only"
  })

  lifecycle {
    prevent_destroy = true
  }
}

#trivy:ignore:AVD-AWS-0089 This is the terminal access-log sink; recursive server-access logging is unsupported.
resource "aws_s3_bucket" "access_logs" {
  bucket        = "${var.bucket_name_prefix}-access-logs"
  force_destroy = false

  tags = merge(var.tags, {
    Purpose = "private-pilot-s3-access-logs"
    Records = "bounded-operational-telemetry"
  })

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_s3_bucket_ownership_controls" "access_logs" {
  bucket = aws_s3_bucket.access_logs.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_public_access_block" "access_logs" {
  bucket                  = aws_s3_bucket.access_logs.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "access_logs" {
  bucket = aws_s3_bucket.access_logs.id
  versioning_configuration {
    status = "Enabled"
  }
}

#trivy:ignore:AVD-AWS-0132 S3 server-access-log destinations require SSE-S3 rather than SSE-KMS.
resource "aws_s3_bucket_server_side_encryption_configuration" "access_logs" {
  bucket = aws_s3_bucket.access_logs.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "access_logs" {
  bucket = aws_s3_bucket.access_logs.id

  rule {
    id     = "bounded-access-log-retention"
    status = "Enabled"
    filter {}

    expiration {
      days = 35
    }

    noncurrent_version_expiration {
      noncurrent_days = 35
    }

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }

  depends_on = [aws_s3_bucket_versioning.access_logs]
}

resource "aws_s3_bucket_policy" "access_logs" {
  bucket = aws_s3_bucket.access_logs.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "DenyInsecureTransport"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource  = [aws_s3_bucket.access_logs.arn, "${aws_s3_bucket.access_logs.arn}/*"]
        Condition = { Bool = { "aws:SecureTransport" = "false" } }
      },
      {
        Sid       = "AllowS3ServerAccessLogs"
        Effect    = "Allow"
        Principal = { Service = "logging.s3.amazonaws.com" }
        Action    = "s3:PutObject"
        Resource  = "${aws_s3_bucket.access_logs.arn}/access/*"
        Condition = {
          ArnLike      = { "aws:SourceArn" = [for bucket in aws_s3_bucket.objects : bucket.arn] }
          StringEquals = { "aws:SourceAccount" = var.aws_account_id }
        }
      },
    ]
  })

  depends_on = [aws_s3_bucket_public_access_block.access_logs]
}

resource "aws_s3_bucket_logging" "objects" {
  for_each = aws_s3_bucket.objects

  bucket        = each.value.id
  target_bucket = aws_s3_bucket.access_logs.id
  target_prefix = "access/${each.key}/"

  depends_on = [aws_s3_bucket_policy.access_logs]
}

resource "aws_s3_bucket_ownership_controls" "objects" {
  for_each = aws_s3_bucket.objects

  bucket = each.value.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_public_access_block" "objects" {
  for_each = aws_s3_bucket.objects

  bucket                  = each.value.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "objects" {
  for_each = aws_s3_bucket.objects

  bucket = each.value.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "objects" {
  for_each = aws_s3_bucket.objects

  bucket = each.value.id
  rule {
    bucket_key_enabled = true
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.data.arn
      sse_algorithm     = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_cors_configuration" "objects" {
  for_each = aws_s3_bucket.objects

  bucket = each.value.id

  cors_rule {
    id              = "private-pilot-browser"
    allowed_methods = each.key == "quarantine" ? ["HEAD", "PUT"] : ["GET", "HEAD"]
    allowed_origins = ["https://${var.hostname}"]
    allowed_headers = ["content-type", "if-none-match", "x-amz-meta-sha256"]
    expose_headers  = ["ETag", "x-amz-version-id"]
    max_age_seconds = 300
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "objects" {
  for_each = aws_s3_bucket.objects

  bucket = each.value.id

  rule {
    id     = "abort-incomplete-uploads"
    status = "Enabled"

    filter {}

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }

  depends_on = [aws_s3_bucket_versioning.objects]
}

resource "aws_ecr_repository" "runtime" {
  for_each = local.runtime_repositories

  name                 = "${var.name}-${each.value}"
  image_tag_mutability = "IMMUTABLE"

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.data.arn
  }

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = merge(var.tags, {
    Component    = each.value
    Architecture = "linux-arm64"
  })

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_ecr_lifecycle_policy" "runtime" {
  for_each = aws_ecr_repository.runtime

  repository = each.value.name
  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Expire untagged release candidates after seven days"
      selection = {
        tagStatus   = "untagged"
        countType   = "sinceImagePushed"
        countUnit   = "days"
        countNumber = 7
      }
      action = { type = "expire" }
    }]
  })
}

resource "aws_iam_role" "guardduty_malware" {
  name_prefix = "${var.name}-guardduty-s3-"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Service = "malware-protection-plan.guardduty.amazonaws.com"
      }
      Action = "sts:AssumeRole"
    }]
  })

  tags = merge(var.tags, { Purpose = "standalone-guardduty-s3-malware-protection" })
}

resource "aws_iam_role_policy" "guardduty_malware" {
  name = "${var.name}-guardduty-s3"
  role = aws_iam_role.guardduty_malware.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "InspectExactQuarantineVersions"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:GetObjectVersion",
          "s3:GetObjectTagging",
          "s3:GetObjectVersionTagging",
          "s3:PutObjectTagging",
          "s3:PutObjectVersionTagging",
        ]
        Resource = "${aws_s3_bucket.objects["quarantine"].arn}/*"
      },
      {
        Sid      = "WriteGuardDutyValidationObject"
        Effect   = "Allow"
        Action   = "s3:PutObject"
        Resource = "${aws_s3_bucket.objects["quarantine"].arn}/malware-protection-resource-validation-object"
      },
      {
        Sid    = "ReadQuarantineConfiguration"
        Effect = "Allow"
        Action = [
          "s3:GetBucketLocation",
          "s3:GetBucketNotification",
          "s3:ListBucket",
          "s3:PutBucketNotification",
        ]
        Resource = aws_s3_bucket.objects["quarantine"].arn
      },
      {
        Sid    = "UseExactDataKey"
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:DescribeKey",
          "kms:GenerateDataKey",
        ]
        Resource = aws_kms_key.data.arn
      },
      {
        Sid    = "ManageGuardDutyOwnedEventRule"
        Effect = "Allow"
        Action = [
          "events:DeleteRule",
          "events:PutRule",
          "events:PutTargets",
          "events:RemoveTargets",
        ]
        Resource = "arn:aws:events:${var.region}:${var.aws_account_id}:rule/DO-NOT-DELETE-AmazonGuardDutyMalwareProtectionS3*"
        Condition = {
          StringLike = {
            "events:ManagedBy" = "malware-protection-plan.guardduty.amazonaws.com"
          }
        }
      },
      {
        Sid    = "InspectGuardDutyOwnedEventRule"
        Effect = "Allow"
        Action = [
          "events:DescribeRule",
          "events:ListTargetsByRule",
        ]
        Resource = "arn:aws:events:${var.region}:${var.aws_account_id}:rule/DO-NOT-DELETE-AmazonGuardDutyMalwareProtectionS3*"
      },
    ]
  })
}

resource "aws_guardduty_malware_protection_plan" "quarantine" {
  role = aws_iam_role.guardduty_malware.arn

  protected_resource {
    s3_bucket {
      bucket_name     = aws_s3_bucket.objects["quarantine"].id
      object_prefixes = ["organizations/"]
    }
  }

  actions {
    tagging {
      status = "ENABLED"
    }
  }

  tags = merge(var.tags, {
    ScanBoundary = "exact-s3-version"
    CleanResult  = "NO_THREATS_FOUND"
  })

  depends_on = [
    aws_iam_role_policy.guardduty_malware,
    aws_s3_bucket_versioning.objects,
  ]

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_vpc_endpoint_policy" "s3" {
  vpc_endpoint_id = aws_vpc_endpoint.s3.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PrivatePilotBucketsOnly"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:*"
        Resource = concat(
          [for bucket in aws_s3_bucket.objects : bucket.arn],
          [for bucket in aws_s3_bucket.objects : "${bucket.arn}/*"],
        )
      },
      {
        Sid       = "PullEcrLayersThroughGatewayEndpoint"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "arn:aws:s3:::prod-${var.region}-starport-layer-bucket/*"
      },
    ]
  })
}
