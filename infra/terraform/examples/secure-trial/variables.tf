variable "name_prefix" {
  description = "Stable resource prefix."
  type        = string
}

variable "environment" {
  description = "Explicit environment identity."
  type        = string
}

variable "region" {
  description = "Explicit approved AWS region."
  type        = string
}

variable "availability_zones" {
  description = "Exactly two approved availability zones."
  type        = list(string)
}

variable "vpc_cidr" {
  description = "Trial VPC CIDR."
  type        = string
}

variable "public_subnet_cidrs" {
  description = "Two public ALB subnet CIDRs."
  type        = list(string)
}

variable "compute_subnet_cidrs" {
  description = "Two private compute subnet CIDRs."
  type        = list(string)
}

variable "database_subnet_cidrs" {
  description = "Two isolated database subnet CIDRs."
  type        = list(string)
}

variable "enable_nat_gateway" {
  description = "Whether private compute uses NAT."
  type        = bool
}

variable "single_nat_gateway" {
  description = "Whether an approved trial accepts one NAT gateway."
  type        = bool
}

variable "certificate_arn" {
  description = "Explicit approved ACM certificate ARN."
  type        = string
}

variable "alarm_topic_arn" {
  description = "Explicit approved alarm topic ARN."
  type        = string
}

variable "otel_endpoint" {
  description = "Private OTel endpoint."
  type        = string
}

variable "bucket_name_prefix" {
  description = "Globally unique approved bucket prefix."
  type        = string
}

variable "repositories" {
  description = "Runtime ECR repositories."
  type        = set(string)
}

variable "secret_names" {
  description = "Secret containers populated outside Terraform."
  type        = set(string)
}

variable "secret_recovery_window_days" {
  description = "Secrets Manager recovery window."
  type        = number
}

variable "application_port" {
  description = "Private application port."
  type        = number
}

variable "ami_id" {
  description = "Explicit immutable AMI ID."
  type        = string
}

variable "image_uri" {
  description = "Private ECR image URI pinned by digest."
  type        = string
}

variable "instance_type" {
  description = "Approved EC2 instance type."
  type        = string
}

variable "min_size" {
  description = "Minimum runtime capacity."
  type        = number
}

variable "desired_capacity" {
  description = "Desired runtime capacity."
  type        = number
}

variable "max_size" {
  description = "Maximum runtime capacity."
  type        = number
}

variable "database_instance_class" {
  description = "Approved RDS instance class."
  type        = string
}

variable "database_engine_version" {
  description = "Approved PostgreSQL engine version."
  type        = string
}

variable "database_allocated_storage" {
  description = "Initial RDS storage in GiB."
  type        = number
}

variable "backup_retention_days" {
  description = "Approved backup retention."
  type        = number
}

variable "log_retention_days" {
  description = "Approved CloudWatch log retention."
  type        = number
}

variable "tags" {
  description = "Mandatory ownership and cost tags."
  type        = map(string)
}
