mock_provider "aws" {}

variables {
  tags = {
    Environment        = "fixture"
    Owner              = "platform-operations"
    CostCenter         = "trial-001"
    DataClassification = "restricted"
    ManagedBy          = "terraform"
  }
}

run "network_has_two_private_availability_zones" {
  command = plan

  module {
    source = "./modules/network"
  }

  variables {
    name                  = "avia-fixture"
    vpc_cidr              = "10.42.0.0/16"
    availability_zones    = ["eu-central-1a", "eu-central-1b"]
    public_subnet_cidrs   = ["10.42.0.0/24", "10.42.1.0/24"]
    compute_subnet_cidrs  = ["10.42.10.0/24", "10.42.11.0/24"]
    database_subnet_cidrs = ["10.42.20.0/24", "10.42.21.0/24"]
    enable_nat_gateway    = false
    single_nat_gateway    = false
  }

  assert {
    condition     = length(aws_subnet.public) == 2
    error_message = "The ALB tier must span exactly two fixture AZs."
  }

  assert {
    condition     = length(aws_subnet.compute) == 2 && length(aws_subnet.database) == 2
    error_message = "Compute and database tiers must each span two private subnets."
  }

  assert {
    condition = alltrue(concat(
      [for subnet in aws_subnet.public : !subnet.map_public_ip_on_launch],
      [for subnet in aws_subnet.compute : !subnet.map_public_ip_on_launch],
      [for subnet in aws_subnet.database : !subnet.map_public_ip_on_launch],
    ))
    error_message = "No instance-facing subnet may auto-assign public addresses."
  }
}

run "security_has_no_ssh_or_wildcard_iam" {
  command = plan

  module {
    source = "./modules/security"
  }

  variables {
    name                = "avia-fixture"
    vpc_id              = "vpc-0123456789abcdef0"
    application_port    = 8443
    database_port       = 5432
    secret_arns         = ["arn:aws:secretsmanager:eu-central-1:111122223333:secret:avia/app"]
    bucket_arns         = ["arn:aws:s3:::avia-fixture-app"]
    kms_key_arns        = ["arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"]
    ecr_repository_arns = ["arn:aws:ecr:eu-central-1:111122223333:repository/avia-api"]
    log_group_arns      = ["arn:aws:logs:eu-central-1:111122223333:log-group:/avia/fixture:*"]
  }

  assert {
    condition     = aws_vpc_security_group_ingress_rule.alb_https.from_port == 443 && aws_vpc_security_group_ingress_rule.alb_https.to_port == 443
    error_message = "The public security group may expose only HTTPS."
  }

  assert {
    condition = alltrue([
      for rule in concat(
        values(aws_vpc_security_group_ingress_rule.application),
        values(aws_vpc_security_group_ingress_rule.database),
      ) : rule.from_port != 22 && rule.to_port != 22
    ])
    error_message = "SSH ingress is forbidden."
  }

  assert {
    condition     = !strcontains(aws_iam_policy.runtime.policy, "\"Action\":\"*\"") && !strcontains(aws_iam_policy.runtime.policy, "\"Resource\":\"*\"")
    error_message = "Runtime IAM policy actions and resources must be scoped."
  }
}

run "service_endpoints_bound_aws_api_egress" {
  command = apply

  module {
    source = "./modules/service-endpoints"
  }

  variables {
    name                          = "avia-fixture"
    region                        = "eu-central-1"
    vpc_id                        = "vpc-0123456789abcdef0"
    private_subnet_ids            = ["subnet-00000000000000011", "subnet-00000000000000012"]
    private_route_table_ids       = ["rtb-00000000000000011", "rtb-00000000000000012"]
    application_security_group_id = "sg-00000000000000002"
  }

  assert {
    condition     = length(aws_vpc_endpoint.interface) == 5
    error_message = "Runtime AWS APIs must use the declared private interface endpoints."
  }

  assert {
    condition     = aws_vpc_security_group_egress_rule.application_interfaces.referenced_security_group_id == aws_security_group.endpoints.id
    error_message = "Application TLS egress must target only the endpoint security group."
  }

  assert {
    condition     = aws_vpc_endpoint.s3.vpc_endpoint_type == "Gateway"
    error_message = "S3 access must use a route-scoped gateway endpoint."
  }
}

run "ecr_is_immutable_encrypted_and_scan_on_push" {
  command = plan

  module {
    source = "./modules/ecr"
  }

  variables {
    name_prefix = "avia-fixture"
    repositories = [
      "gateway",
      "web",
      "api",
      "worker",
      "scheduler",
      "keycloak",
    ]
    kms_key_arn = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
  }

  assert {
    condition     = alltrue([for repository in aws_ecr_repository.this : repository.image_tag_mutability == "IMMUTABLE"])
    error_message = "Every ECR repository must reject mutable tags."
  }

  assert {
    condition     = alltrue([for repository in aws_ecr_repository.this : repository.image_scanning_configuration[0].scan_on_push])
    error_message = "Every ECR repository must scan on push."
  }

  assert {
    condition     = alltrue([for repository in aws_ecr_repository.this : repository.encryption_configuration[0].encryption_type == "KMS"])
    error_message = "Every ECR repository must use KMS encryption."
  }
}

run "load_balancer_is_https_only" {
  command = plan

  module {
    source = "./modules/load-balancer"
  }

  variables {
    name              = "avia-fixture"
    vpc_id            = "vpc-0123456789abcdef0"
    public_subnet_ids = ["subnet-00000000000000001", "subnet-00000000000000002"]
    security_group_id = "sg-00000000000000001"
    certificate_arn   = "arn:aws:acm:eu-central-1:111122223333:certificate/11111111-2222-3333-4444-555555555555"
    target_port       = 8443
  }

  assert {
    condition     = !aws_lb.this.internal && aws_lb.this.load_balancer_type == "application"
    error_message = "The public entrypoint must be an internet-facing ALB."
  }

  assert {
    condition     = aws_lb_listener.https.port == 443 && aws_lb_listener.https.protocol == "HTTPS"
    error_message = "The ALB listener must be HTTPS-only."
  }

  assert {
    condition     = aws_lb_listener.https.certificate_arn == "arn:aws:acm:eu-central-1:111122223333:certificate/11111111-2222-3333-4444-555555555555"
    error_message = "The listener must use the explicitly supplied certificate."
  }
}

run "compute_is_private_digest_bound_and_imdsv2_only" {
  command = plan

  module {
    source = "./modules/compute"
  }

  variables {
    name                  = "avia-fixture"
    region                = "eu-central-1"
    private_subnet_ids    = ["subnet-00000000000000011", "subnet-00000000000000012"]
    security_group_ids    = ["sg-00000000000000002"]
    instance_profile_name = "avia-fixture-runtime"
    instance_type         = "t4g.medium"
    ami_id                = "ami-0123456789abcdef0"
    image_uri             = "111122223333.dkr.ecr.eu-central-1.amazonaws.com/avia-api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    secret_arns           = ["arn:aws:secretsmanager:eu-central-1:111122223333:secret:avia/app"]
    kms_key_arn           = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
    target_group_arns     = ["arn:aws:elasticloadbalancing:eu-central-1:111122223333:targetgroup/avia/1111111111111111"]
    otel_endpoint         = "http://127.0.0.1:4318"
    min_size              = 2
    desired_capacity      = 2
    max_size              = 4
  }

  assert {
    condition     = aws_launch_template.this.metadata_options[0].http_tokens == "required" && aws_launch_template.this.metadata_options[0].http_endpoint == "enabled"
    error_message = "Launch templates must require IMDSv2."
  }

  assert {
    condition     = !aws_launch_template.this.network_interfaces[0].associate_public_ip_address
    error_message = "Compute instances must not receive public addresses."
  }

  assert {
    condition     = aws_launch_template.this.block_device_mappings[0].ebs[0].encrypted && aws_launch_template.this.block_device_mappings[0].ebs[0].kms_key_id != null
    error_message = "Root EBS must use the supplied KMS key."
  }

  assert {
    condition     = strcontains(base64decode(aws_launch_template.this.user_data), "@sha256:") && !strcontains(base64decode(aws_launch_template.this.user_data), "secret_value")
    error_message = "User data must use an image digest and secret references only."
  }
}

run "database_is_private_encrypted_and_protected" {
  command = plan

  module {
    source = "./modules/database"
  }

  variables {
    name                  = "avia-fixture"
    subnet_ids            = ["subnet-00000000000000021", "subnet-00000000000000022"]
    security_group_ids    = ["sg-00000000000000003"]
    kms_key_arn           = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
    instance_class        = "db.t4g.medium"
    engine_version        = "17.6"
    allocated_storage     = 50
    backup_retention_days = 7
    deletion_protection   = true
  }

  assert {
    condition     = !aws_db_instance.this.publicly_accessible && aws_db_instance.this.multi_az
    error_message = "RDS must be private and multi-AZ."
  }

  assert {
    condition     = aws_db_instance.this.storage_encrypted && aws_db_instance.this.kms_key_id != null
    error_message = "RDS storage must use the supplied KMS key."
  }

  assert {
    condition     = aws_db_instance.this.deletion_protection && aws_db_instance.this.backup_retention_period >= 7 && aws_db_instance.this.skip_final_snapshot == false
    error_message = "RDS must retain backups and require a final snapshot."
  }
}

run "object_storage_is_private_versioned_and_retained" {
  command = plan

  module {
    source = "./modules/object-storage"
  }

  variables {
    name_prefix           = "avia-fixture-111122223333"
    kms_key_arn           = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
    backup_retention_days = 30
  }

  assert {
    condition = alltrue([
      aws_s3_bucket_public_access_block.application.block_public_acls,
      aws_s3_bucket_public_access_block.application.block_public_policy,
      aws_s3_bucket_public_access_block.application.ignore_public_acls,
      aws_s3_bucket_public_access_block.application.restrict_public_buckets,
      aws_s3_bucket_public_access_block.backup.block_public_acls,
      aws_s3_bucket_public_access_block.backup.block_public_policy,
      aws_s3_bucket_public_access_block.backup.ignore_public_acls,
      aws_s3_bucket_public_access_block.backup.restrict_public_buckets,
    ])
    error_message = "Application and backup buckets must block all public access."
  }

  assert {
    condition     = aws_s3_bucket_versioning.application.versioning_configuration[0].status == "Enabled" && aws_s3_bucket_versioning.backup.versioning_configuration[0].status == "Enabled"
    error_message = "Application and backup buckets must be versioned."
  }

  assert {
    condition     = aws_s3_bucket.backup.object_lock_enabled
    error_message = "The backup bucket must enable object lock at creation."
  }
}

run "identity_secrets_are_kms_encrypted_references" {
  command = apply

  module {
    source = "./modules/identity-secrets"
  }

  variables {
    name_prefix = "avia-fixture"
    secret_names = [
      "database-url",
      "oidc-client-secret",
      "session-encryption-key",
    ]
    recovery_window_days = 14
  }

  assert {
    condition     = aws_kms_key.secrets.enable_key_rotation
    error_message = "The secrets KMS key must rotate."
  }

  assert {
    condition     = alltrue([for secret in aws_secretsmanager_secret.this : secret.kms_key_id != null && secret.recovery_window_in_days >= 7])
    error_message = "Secret containers must be KMS encrypted and recoverable."
  }
}

run "observability_logs_are_encrypted_and_bounded" {
  command = plan

  module {
    source = "./modules/observability"
  }

  variables {
    name_prefix        = "avia-fixture"
    kms_key_arn        = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
    log_retention_days = 30
    alarm_topic_arn    = "arn:aws:sns:eu-central-1:111122223333:avia-alerts"
    otel_endpoint      = "http://127.0.0.1:4318"
  }

  assert {
    condition     = alltrue([for group in aws_cloudwatch_log_group.this : group.kms_key_id != null && group.retention_in_days == 30])
    error_message = "Every operational log group must be encrypted and bounded."
  }
}

run "backup_vault_and_plan_are_encrypted_and_retained" {
  command = plan

  module {
    source = "./modules/backup"
  }

  variables {
    name_prefix           = "avia-fixture"
    kms_key_arn           = "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555"
    resource_arns         = ["arn:aws:rds:eu-central-1:111122223333:db:avia-fixture"]
    backup_retention_days = 30
  }

  assert {
    condition     = aws_backup_vault.this.kms_key_arn != null
    error_message = "The backup vault must use the supplied KMS key."
  }

  assert {
    condition = one(
      one(aws_backup_plan.this.rule).lifecycle
    ).delete_after >= 30
    error_message = "The backup plan must retain recovery points for the approved period."
  }
}
