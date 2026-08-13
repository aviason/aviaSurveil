variable "name_prefix" {
  description = "Stable name prefix for operational telemetry."
  type        = string
}

variable "kms_key_arn" {
  description = "KMS key for CloudWatch logs."
  type        = string
}

variable "log_retention_days" {
  description = "Bounded CloudWatch log retention."
  type        = number

  validation {
    condition     = contains([7, 14, 30, 60, 90, 120, 150, 180, 365], var.log_retention_days)
    error_message = "log_retention_days must use a supported bounded CloudWatch value."
  }
}

variable "alarm_topic_arn" {
  description = "Approved private alarm notification topic."
  type        = string
}

variable "otel_endpoint" {
  description = "Private OTLP collector endpoint consumed by runtime bootstrap."
  type        = string

  validation {
    condition     = can(regex("^https?://", var.otel_endpoint))
    error_message = "otel_endpoint must be an explicit HTTP(S) endpoint."
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

locals {
  log_groups = toset(["application", "auth", "gateway", "worker"])
}

resource "aws_cloudwatch_log_group" "this" {
  for_each = local.log_groups

  name              = "/aviasurveil360/${var.name_prefix}/${each.value}"
  kms_key_id        = var.kms_key_arn
  retention_in_days = var.log_retention_days

  tags = merge(var.tags, { SignalOwner = each.value })
}

resource "aws_ssm_parameter" "otel_endpoint" {
  name        = "/aviasurveil360/${var.name_prefix}/otel-endpoint"
  description = "Private OTLP endpoint reference"
  type        = "String"
  value       = var.otel_endpoint

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "api_errors" {
  alarm_name          = "${var.name_prefix}-api-errors"
  alarm_description   = "Candidate API 5xx rate exceeded the reviewed threshold"
  namespace           = "AviaSurveil360"
  metric_name         = "http.server.errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  datapoints_to_alarm = 2
  period              = 300
  statistic           = "Sum"
  threshold           = 5
  treat_missing_data  = "notBreaching"
  alarm_actions       = [var.alarm_topic_arn]
  ok_actions          = [var.alarm_topic_arn]

  dimensions = {
    Environment = var.tags["Environment"]
    Service     = "api"
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "outbox_age" {
  alarm_name          = "${var.name_prefix}-outbox-ready-age"
  alarm_description   = "Oldest candidate outbox work exceeded ten minutes"
  namespace           = "AviaSurveil360"
  metric_name         = "outbox.ready.age"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  datapoints_to_alarm = 2
  period              = 300
  statistic           = "Maximum"
  threshold           = 600
  treat_missing_data  = "notBreaching"
  alarm_actions       = [var.alarm_topic_arn]
  ok_actions          = [var.alarm_topic_arn]

  dimensions = {
    Environment = var.tags["Environment"]
    Service     = "worker"
  }

  tags = var.tags
}

output "log_group_arns" {
  description = "Operational log group ARNs keyed by service."
  value       = { for name, group in aws_cloudwatch_log_group.this : name => group.arn }
}

output "otel_parameter_arn" {
  description = "SSM parameter ARN containing the non-secret OTLP endpoint."
  value       = aws_ssm_parameter.otel_endpoint.arn
}
