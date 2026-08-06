variable "name" {
  description = "Stable budget name prefix."
  type        = string
}

variable "monthly_ceiling_usd" {
  description = "Owner-approved monthly spend ceiling including non-compute costs."
  type        = number

  validation {
    condition     = var.monthly_ceiling_usd > 0 && var.monthly_ceiling_usd <= 500
    error_message = "monthly_ceiling_usd must be positive and bounded at 500 USD."
  }
}

variable "one_run_ceiling_usd" {
  description = "Owner-approved one-run spend ceiling."
  type        = number

  validation {
    condition     = var.one_run_ceiling_usd > 0 && var.one_run_ceiling_usd <= 250
    error_message = "one_run_ceiling_usd must be positive and bounded at 250 USD."
  }
}

variable "estimated_monthly_usd" {
  description = "Current reviewed monthly estimate."
  type        = number
}

variable "estimated_one_run_usd" {
  description = "Current reviewed one-run estimate."
  type        = number
}

variable "trial_expiry" {
  description = "ISO-8601 trial expiry bound used in the budget notification."
  type        = string

  validation {
    condition     = can(formatdate("YYYY-MM-DD", var.trial_expiry))
    error_message = "trial_expiry must be an ISO-8601 timestamp."
  }
}

variable "alert_recipients" {
  description = "Owner-controlled budget alert email recipients."
  type        = set(string)

  validation {
    condition     = length(var.alert_recipients) > 0 && alltrue([for recipient in var.alert_recipients : can(regex("^[^@[:space:]]+@[^@[:space:]]+\\.[^@[:space:]]+$", recipient))])
    error_message = "alert_recipients must contain at least one valid owner email."
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
    error_message = "tags must include non-empty Environment, Owner, CostCenter, DataClassification, and ManagedBy values."
  }
}

check "cost_estimate_is_within_owner_ceiling" {
  assert {
    condition     = var.estimated_monthly_usd > 0 && var.estimated_monthly_usd <= var.monthly_ceiling_usd && var.estimated_one_run_usd > 0 && var.estimated_one_run_usd <= var.one_run_ceiling_usd
    error_message = "The reviewed cost estimate must be positive and below both owner ceilings."
  }
}

resource "aws_budgets_budget" "trial" {
  name         = "${var.name}-monthly"
  budget_type  = "COST"
  limit_amount = tostring(var.monthly_ceiling_usd)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  cost_filter {
    name   = "TagKeyValue"
    values = ["user:TrialProfile$aws-ipv6-trial"]
  }

  notification {
    comparison_operator        = "GREATER_THAN"
    threshold                  = 80
    threshold_type             = "PERCENTAGE"
    notification_type          = "FORECASTED"
    subscriber_email_addresses = tolist(var.alert_recipients)
  }

  notification {
    comparison_operator        = "GREATER_THAN"
    threshold                  = 100
    threshold_type             = "PERCENTAGE"
    notification_type          = "ACTUAL"
    subscriber_email_addresses = tolist(var.alert_recipients)
  }

  tags = merge(var.tags, {
    Name             = "${var.name}-monthly"
    TrialProfile     = "aws-ipv6-trial"
    TrialExpiry      = var.trial_expiry
    OneRunCeilingUsd = tostring(var.one_run_ceiling_usd)
    EstimatedMonthly = tostring(var.estimated_monthly_usd)
    EstimatedOneRun  = tostring(var.estimated_one_run_usd)
  })
}

output "budget_name" {
  description = "AWS Budget name."
  value       = aws_budgets_budget.trial.name
}

output "monthly_ceiling_usd" {
  description = "Bounded owner-approved monthly ceiling."
  value       = var.monthly_ceiling_usd
}
