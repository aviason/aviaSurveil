output "topology" {
  description = "Literal non-HA and no-public-IPv4 topology contract for offline review."
  value = {
    architecture                   = "linux/arm64"
    availability                   = "single-workload-az-non-ha"
    runtime_instances              = 1
    runtime_instance_type          = "t4g.small"
    runtime_security_group_ingress = 0
    runtime_public_ipv4_addresses  = 0
    runtime_global_ipv6_addresses  = 1
    application_load_balancers     = 0
    nat_gateways                   = 0
    internet_gateways              = 0
    egress_only_internet_gateways  = 1
    public_subnets                 = 0
    dual_stack_app_subnets         = 1
    private_db_subnets             = 2
    rds_instances                  = 1
    rds_instance_class             = "db.t4g.micro"
    rds_multi_az                   = false
    interface_endpoints            = 0
    s3_gateway_endpoints           = 1
    cloudflare_tunnels             = 1
    cloudflare_dns_records         = var.cloudflare_dns_cutover_authorized ? 1 : 0
  }
}

output "cloudflare_tunnel" {
  description = "Outbound-only Cloudflare Tunnel identity and non-secret routing contract."
  value = {
    account_id                 = var.cloudflare_account_id
    tunnel_id                  = cloudflare_zero_trust_tunnel_cloudflared.this.id
    tunnel_name                = cloudflare_zero_trust_tunnel_cloudflared.this.name
    hostname                   = var.hostname
    dns_cutover_authorized     = var.cloudflare_dns_cutover_authorized
    dns_record_id              = try(cloudflare_dns_record.application[var.hostname].id, null)
    origin                     = "http://127.0.0.1:8080"
    connector_parameter_arn    = aws_ssm_parameter.cloudflare_connector.arn
    connector_token_in_state   = false
    connector_token_authorized = false
  }
}

output "runtime_instance_id" {
  description = "The sole private ARM64 runtime instance."
  value       = aws_instance.runtime.id
}

output "runtime_instance_ipv6_address" {
  description = "The host global IPv6 address; its security group has zero ingress rules."
  value       = aws_instance.runtime.ipv6_addresses[0]
}

output "runtime_role_arn" {
  description = "Least-privilege EC2 instance role."
  value       = aws_iam_role.runtime.arn
}

output "database" {
  description = "Single-AZ RDS endpoint and bootstrap-only managed master secret reference."
  sensitive   = true
  value = {
    arn               = aws_db_instance.this.arn
    endpoint          = aws_db_instance.this.endpoint
    master_secret_arn = try(aws_db_instance.this.master_user_secret[0].secret_arn, null)
    logical_databases = ["aviasurveil360", "keycloak"]
  }
}

output "bucket_arns" {
  description = "Private versioned S3 buckets keyed by purpose."
  value       = { for purpose, bucket in aws_s3_bucket.objects : purpose => bucket.arn }
}

output "ecr_repository_urls" {
  description = "IPv6-capable immutable ECR repositories keyed by runtime artifact."
  value       = { for component, repository in aws_ecr_repository.runtime : component => "${var.aws_account_id}.dkr-ecr.${var.region}.on.aws/${repository.name}" }
}

output "runtime_secret_arns" {
  description = "Ten empty secret-container references; no value is managed by this module."
  value       = { for name, secret in aws_secretsmanager_secret.runtime : name => secret.arn }
}

output "guardduty_malware_protection_plan_id" {
  description = "Standalone exact-version quarantine malware protection plan."
  value       = aws_guardduty_malware_protection_plan.quarantine.id
}

output "backup_vault_arn" {
  description = "Encrypted 35-day engineering recovery vault."
  value       = aws_backup_vault.this.arn
}

output "budget_name" {
  description = "Mandatory private-pilot AWS Budget."
  value       = aws_budgets_budget.this.name
}

output "alert_topic_arn" {
  description = "Encrypted same-account SNS target for CloudWatch and AWS Budget notifications."
  value       = aws_sns_topic.alerts.arn
}
