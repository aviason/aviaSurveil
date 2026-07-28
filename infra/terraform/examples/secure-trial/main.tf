terraform {
  required_version = ">= 1.10, < 2.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 6.0, < 7.0"
    }
  }
}

provider "aws" {
  region = var.region

  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_region_validation      = true
  skip_requesting_account_id  = true
}

locals {
  name = "${var.name_prefix}-${var.environment}"
}

module "identity_secrets" {
  source = "../../modules/identity-secrets"

  name_prefix          = local.name
  secret_names         = var.secret_names
  recovery_window_days = var.secret_recovery_window_days
  tags                 = var.tags
}

module "network" {
  source = "../../modules/network"

  name                  = local.name
  vpc_cidr              = var.vpc_cidr
  availability_zones    = var.availability_zones
  public_subnet_cidrs   = var.public_subnet_cidrs
  compute_subnet_cidrs  = var.compute_subnet_cidrs
  database_subnet_cidrs = var.database_subnet_cidrs
  enable_nat_gateway    = var.enable_nat_gateway
  single_nat_gateway    = var.single_nat_gateway
  tags                  = var.tags
}

module "observability" {
  source = "../../modules/observability"

  name_prefix        = local.name
  kms_key_arn        = module.identity_secrets.kms_key_arn
  log_retention_days = var.log_retention_days
  alarm_topic_arn    = var.alarm_topic_arn
  otel_endpoint      = var.otel_endpoint
  tags               = var.tags
}

module "object_storage" {
  source = "../../modules/object-storage"

  name_prefix           = var.bucket_name_prefix
  kms_key_arn           = module.identity_secrets.kms_key_arn
  backup_retention_days = var.backup_retention_days
  tags                  = var.tags
}

module "ecr" {
  source = "../../modules/ecr"

  name_prefix  = local.name
  repositories = var.repositories
  kms_key_arn  = module.identity_secrets.kms_key_arn
  tags         = var.tags
}

module "security" {
  source = "../../modules/security"

  name                = local.name
  vpc_id              = module.network.vpc_id
  application_port    = var.application_port
  database_port       = 5432
  secret_arns         = values(module.identity_secrets.secret_arns)
  bucket_arns         = [module.object_storage.application_bucket_arn, module.object_storage.backup_bucket_arn]
  kms_key_arns        = [module.identity_secrets.kms_key_arn]
  ecr_repository_arns = values(module.ecr.repository_arns)
  log_group_arns      = values(module.observability.log_group_arns)
  tags                = var.tags
}

module "service_endpoints" {
  source = "../../modules/service-endpoints"

  name                          = local.name
  region                        = var.region
  vpc_id                        = module.network.vpc_id
  private_subnet_ids            = module.network.private_compute_subnet_ids
  private_route_table_ids       = module.network.private_compute_route_table_ids
  application_security_group_id = module.security.application_security_group_id
  tags                          = var.tags
}

module "load_balancer" {
  source = "../../modules/load-balancer"

  name              = local.name
  vpc_id            = module.network.vpc_id
  public_subnet_ids = module.network.public_subnet_ids
  security_group_id = module.security.alb_security_group_id
  certificate_arn   = var.certificate_arn
  target_port       = var.application_port
  tags              = var.tags
}

module "database" {
  source = "../../modules/database"

  name                  = local.name
  subnet_ids            = module.network.private_database_subnet_ids
  security_group_ids    = [module.security.database_security_group_id]
  kms_key_arn           = module.identity_secrets.kms_key_arn
  instance_class        = var.database_instance_class
  engine_version        = var.database_engine_version
  allocated_storage     = var.database_allocated_storage
  backup_retention_days = var.backup_retention_days
  deletion_protection   = true
  tags                  = var.tags
}

module "compute" {
  source = "../../modules/compute"

  name                  = local.name
  region                = var.region
  private_subnet_ids    = module.network.private_compute_subnet_ids
  security_group_ids    = [module.security.application_security_group_id]
  instance_profile_name = module.security.instance_profile_name
  instance_type         = var.instance_type
  ami_id                = var.ami_id
  image_uri             = var.image_uri
  secret_arns           = values(module.identity_secrets.secret_arns)
  kms_key_arn           = module.identity_secrets.kms_key_arn
  target_group_arns     = [module.load_balancer.target_group_arn]
  otel_endpoint         = var.otel_endpoint
  min_size              = var.min_size
  desired_capacity      = var.desired_capacity
  max_size              = var.max_size
  tags                  = var.tags

  depends_on = [module.service_endpoints]
}

module "backup" {
  source = "../../modules/backup"

  name_prefix           = local.name
  kms_key_arn           = module.identity_secrets.kms_key_arn
  resource_arns         = [module.database.database_arn, module.object_storage.application_bucket_arn]
  backup_retention_days = var.backup_retention_days
  tags                  = var.tags
}

output "dependency_contract" {
  description = "Dependency outputs consumed by the Terragrunt composition."
  value = {
    vpc_id                 = module.network.vpc_id
    public_subnet_ids      = module.network.public_subnet_ids
    compute_subnet_ids     = module.network.private_compute_subnet_ids
    database_subnet_ids    = module.network.private_database_subnet_ids
    repository_urls        = module.ecr.repository_urls
    application_bucket_arn = module.object_storage.application_bucket_arn
    backup_bucket_arn      = module.object_storage.backup_bucket_arn
    database_arn           = module.database.database_arn
    database_endpoint      = module.database.endpoint
    load_balancer_dns_name = module.load_balancer.dns_name
    autoscaling_group_name = module.compute.autoscaling_group_name
    backup_vault_arn       = module.backup.vault_arn
  }
}
