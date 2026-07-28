variable "name" {
  description = "Stable name prefix for private service endpoints."
  type        = string
}

variable "region" {
  description = "Explicit approved AWS region."
  type        = string
}

variable "vpc_id" {
  description = "VPC identifier."
  type        = string
}

variable "private_subnet_ids" {
  description = "Private compute subnets used by interface endpoints."
  type        = list(string)

  validation {
    condition     = length(var.private_subnet_ids) >= 2
    error_message = "private_subnet_ids must span at least two subnets."
  }
}

variable "private_route_table_ids" {
  description = "Private compute route tables used by the S3 gateway endpoint."
  type        = list(string)

  validation {
    condition     = length(var.private_route_table_ids) >= 2
    error_message = "private_route_table_ids must include both compute routes."
  }
}

variable "application_security_group_id" {
  description = "Application security group receiving endpoint-scoped egress."
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

locals {
  interface_services = toset([
    "ecr.api",
    "ecr.dkr",
    "logs",
    "secretsmanager",
    "ssm",
  ])
}

resource "aws_security_group" "endpoints" {
  name_prefix = "${var.name}-endpoints-"
  description = "Private AWS service endpoints"
  vpc_id      = var.vpc_id

  tags = merge(var.tags, { Name = "${var.name}-endpoints" })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_ingress_rule" "endpoint_https" {
  security_group_id            = aws_security_group.endpoints.id
  description                  = "TLS from private application instances"
  referenced_security_group_id = var.application_security_group_id
  from_port                    = 443
  ip_protocol                  = "tcp"
  to_port                      = 443
}

resource "aws_vpc_endpoint" "interface" {
  for_each = local.interface_services

  vpc_id              = var.vpc_id
  service_name        = "com.amazonaws.${var.region}.${each.value}"
  vpc_endpoint_type   = "Interface"
  private_dns_enabled = true
  subnet_ids          = var.private_subnet_ids
  security_group_ids  = [aws_security_group.endpoints.id]

  tags = merge(var.tags, { Name = "${var.name}-${replace(each.value, ".", "-")}" })
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id            = var.vpc_id
  service_name      = "com.amazonaws.${var.region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = var.private_route_table_ids

  tags = merge(var.tags, { Name = "${var.name}-s3" })
}

resource "aws_vpc_security_group_egress_rule" "application_interfaces" {
  security_group_id            = var.application_security_group_id
  description                  = "TLS to private AWS interface endpoints"
  referenced_security_group_id = aws_security_group.endpoints.id
  from_port                    = 443
  ip_protocol                  = "tcp"
  to_port                      = 443
}

resource "aws_vpc_security_group_egress_rule" "application_s3" {
  security_group_id = var.application_security_group_id
  description       = "TLS to the regional S3 gateway endpoint"
  prefix_list_id    = aws_vpc_endpoint.s3.prefix_list_id
  from_port         = 443
  ip_protocol       = "tcp"
  to_port           = 443
}

output "interface_endpoint_ids" {
  description = "Interface endpoint IDs keyed by AWS service."
  value       = { for service, endpoint in aws_vpc_endpoint.interface : service => endpoint.id }
}

output "s3_endpoint_id" {
  description = "S3 gateway endpoint ID."
  value       = aws_vpc_endpoint.s3.id
}
