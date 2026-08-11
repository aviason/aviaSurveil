locals {
  runtime_log_groups = toset([
    "host",
    "cloudflared",
    "gateway",
    "api",
    "worker",
    "keycloak",
    "vpc-flow",
  ])
}

resource "aws_sns_topic" "alerts" {
  name              = "${var.name}-alerts"
  display_name      = "AviaSurveil360 private pilot alerts"
  kms_master_key_id = "alias/aws/sns"
  signature_version = 2

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AccountOwnerManagement"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::${var.aws_account_id}:root" }
        Action = [
          "SNS:GetTopicAttributes",
          "SNS:ListSubscriptionsByTopic",
          "SNS:Publish",
          "SNS:SetTopicAttributes",
          "SNS:Subscribe",
        ]
        Resource = "arn:aws:sns:${var.region}:${var.aws_account_id}:${var.name}-alerts"
      },
      {
        Sid    = "CloudWatchAndBudgetsPublish"
        Effect = "Allow"
        Principal = {
          Service = ["budgets.amazonaws.com", "cloudwatch.amazonaws.com"]
        }
        Action   = "SNS:Publish"
        Resource = "arn:aws:sns:${var.region}:${var.aws_account_id}:${var.name}-alerts"
        Condition = {
          StringEquals = { "AWS:SourceAccount" = var.aws_account_id }
        }
      },
    ]
  })

  tags = merge(var.tags, { SignalOwner = "platform-operations" })

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_cloudwatch_log_group" "runtime" {
  for_each = local.runtime_log_groups

  name              = "/aviasurveil360/${var.name}/${each.value}"
  kms_key_id        = aws_kms_key.logs.arn
  retention_in_days = 30

  tags = merge(var.tags, { SignalOwner = each.value })

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_cloudwatch_metric_alarm" "ec2_cpu" {
  alarm_name          = "${var.name}-ec2-cpu"
  alarm_description   = "The one t4g.small sustained high CPU"
  namespace           = "AWS/EC2"
  metric_name         = "CPUUtilization"
  dimensions          = { InstanceId = aws_instance.runtime.id }
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  datapoints_to_alarm = 3
  period              = 300
  statistic           = "Average"
  threshold           = 75
  treat_missing_data  = "breaching"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "ec2_cpu_credit_balance" {
  alarm_name          = "${var.name}-ec2-cpu-credit-balance"
  alarm_description   = "The T4g CPU credit balance is below the pilot floor"
  namespace           = "AWS/EC2"
  metric_name         = "CPUCreditBalance"
  dimensions          = { InstanceId = aws_instance.runtime.id }
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 2
  datapoints_to_alarm = 2
  period              = 300
  statistic           = "Minimum"
  threshold           = 20
  treat_missing_data  = "breaching"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "ec2_status" {
  alarm_name          = "${var.name}-ec2-status"
  alarm_description   = "The sole runtime host failed an instance or system status check"
  namespace           = "AWS/EC2"
  metric_name         = "StatusCheckFailed"
  dimensions          = { InstanceId = aws_instance.runtime.id }
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  period              = 60
  statistic           = "Maximum"
  threshold           = 0
  treat_missing_data  = "breaching"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "rds_freeable_memory" {
  alarm_name          = "${var.name}-rds-freeable-memory"
  alarm_description   = "The db.t4g.micro freeable memory is below 128 MiB"
  namespace           = "AWS/RDS"
  metric_name         = "FreeableMemory"
  dimensions          = { DBInstanceIdentifier = aws_db_instance.this.identifier }
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 3
  datapoints_to_alarm = 3
  period              = 300
  statistic           = "Average"
  threshold           = 134217728
  treat_missing_data  = "breaching"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "rds_connections" {
  alarm_name          = "${var.name}-rds-connections"
  alarm_description   = "Combined application and Keycloak connection use exceeded the bounded pool budget"
  namespace           = "AWS/RDS"
  metric_name         = "DatabaseConnections"
  dimensions          = { DBInstanceIdentifier = aws_db_instance.this.identifier }
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  datapoints_to_alarm = 2
  period              = 300
  statistic           = "Maximum"
  threshold           = 40
  treat_missing_data  = "breaching"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "rds_storage" {
  alarm_name          = "${var.name}-rds-free-storage"
  alarm_description   = "The one RDS instance has less than 4 GiB free storage"
  namespace           = "AWS/RDS"
  metric_name         = "FreeStorageSpace"
  dimensions          = { DBInstanceIdentifier = aws_db_instance.this.identifier }
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 2
  datapoints_to_alarm = 2
  period              = 300
  statistic           = "Minimum"
  threshold           = 4294967296
  treat_missing_data  = "breaching"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "cloudflare_tunnel_connections" {
  alarm_name          = "${var.name}-cloudflare-tunnel-connections"
  alarm_description   = "The sole connector reports fewer than four Cloudflare Tunnel edge connections"
  namespace           = "AviaSurveil360/PrivatePilot"
  metric_name         = "CloudflaredTunnelHAConnections"
  dimensions          = { InstanceId = aws_instance.runtime.id }
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 2
  datapoints_to_alarm = 2
  period              = 60
  statistic           = "Minimum"
  threshold           = 4
  treat_missing_data  = "breaching"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
  tags                = var.tags
}

resource "aws_backup_vault" "this" {
  name        = "${var.name}-vault"
  kms_key_arn = aws_kms_key.data.arn
  tags        = var.tags

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_backup_plan" "this" {
  name = "${var.name}-daily"

  rule {
    rule_name         = "daily-35-day-engineering-recovery"
    target_vault_name = aws_backup_vault.this.name
    schedule          = "cron(0 2 * * ? *)"
    start_window      = 60
    completion_window = 360

    lifecycle {
      delete_after = 35
    }

    recovery_point_tags = merge(var.tags, { RecoveryClass = "private-pilot-35-day" })
  }

  tags = var.tags
}

resource "aws_iam_role" "backup" {
  name_prefix = "${var.name}-backup-"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "backup.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "backup" {
  role       = aws_iam_role.backup.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSBackupServiceRolePolicyForBackup"
}

resource "aws_iam_role_policy_attachment" "backup_s3" {
  role       = aws_iam_role.backup.name
  policy_arn = "arn:aws:iam::aws:policy/AWSBackupServiceRolePolicyForS3Backup"
}

resource "aws_iam_role_policy_attachment" "restore_s3" {
  role       = aws_iam_role.backup.name
  policy_arn = "arn:aws:iam::aws:policy/AWSBackupServiceRolePolicyForS3Restore"
}

resource "aws_backup_selection" "this" {
  name         = "${var.name}-coordinated-rds-and-s3"
  iam_role_arn = aws_iam_role.backup.arn
  plan_id      = aws_backup_plan.this.id
  resources = concat(
    [aws_db_instance.this.arn],
    [for bucket in aws_s3_bucket.objects : bucket.arn],
  )
}

resource "aws_budgets_budget" "this" {
  name         = "${var.name}-monthly"
  budget_type  = "COST"
  limit_amount = tostring(var.budget_monthly_usd)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  notification {
    comparison_operator       = "GREATER_THAN"
    threshold                 = 80
    threshold_type            = "PERCENTAGE"
    notification_type         = "FORECASTED"
    subscriber_sns_topic_arns = [aws_sns_topic.alerts.arn]
  }

  notification {
    comparison_operator       = "GREATER_THAN"
    threshold                 = 100
    threshold_type            = "PERCENTAGE"
    notification_type         = "ACTUAL"
    subscriber_sns_topic_arns = [aws_sns_topic.alerts.arn]
  }

  tags = merge(var.tags, {
    Name         = "${var.name}-monthly"
    PilotProfile = "aws-private-pilot"
  })

  lifecycle {
    prevent_destroy = true
  }
}
