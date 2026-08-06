variable "name" {
  description = "Stable name prefix for the disposable IPv6 trial."
  type        = string
}

variable "vpc_ipv4_cidr" {
  description = "AWS-required private IPv4 CIDR retained on the dual-stack VPC."
  type        = string

  validation {
    condition     = can(cidrnetmask(var.vpc_ipv4_cidr)) && !strcontains(var.vpc_ipv4_cidr, ":")
    error_message = "vpc_ipv4_cidr must be a valid IPv4 CIDR."
  }
}

variable "availability_zone" {
  description = "One owner-approved availability zone."
  type        = string
}

variable "cloudflare_tunnel_ipv6_cidrs" {
  description = "Reviewed Cloudflare Tunnel IPv6 egress ranges for TCP/UDP 7844."
  type        = set(string)

  validation {
    condition     = length(var.cloudflare_tunnel_ipv6_cidrs) > 0 && alltrue([for cidr in var.cloudflare_tunnel_ipv6_cidrs : can(cidrnetmask(cidr)) && strcontains(cidr, ":") && cidr != "::/0"])
    error_message = "cloudflare_tunnel_ipv6_cidrs must contain non-default IPv6 CIDRs."
  }
}

variable "management_ipv6_cidrs" {
  description = "Reviewed IPv6 AWS Systems Manager endpoint ranges for HTTPS."
  type        = set(string)

  validation {
    condition     = length(var.management_ipv6_cidrs) > 0 && alltrue([for cidr in var.management_ipv6_cidrs : can(cidrnetmask(cidr)) && strcontains(cidr, ":") && cidr != "::/0"])
    error_message = "management_ipv6_cidrs must contain non-default IPv6 CIDRs."
  }
}

variable "dns_ipv6_cidrs" {
  description = "Reviewed IPv6 DNS resolver ranges."
  type        = set(string)

  validation {
    condition     = length(var.dns_ipv6_cidrs) > 0 && alltrue([for cidr in var.dns_ipv6_cidrs : can(cidrnetmask(cidr)) && strcontains(cidr, ":") && cidr != "::/0"])
    error_message = "dns_ipv6_cidrs must contain non-default IPv6 CIDRs."
  }
}

variable "bootstrap_https_ipv6_cidrs" {
  description = "Time-bounded reviewed IPv6 HTTPS destinations used during bootstrap and image delivery."
  type        = set(string)

  validation {
    condition     = length(var.bootstrap_https_ipv6_cidrs) > 0 && alltrue([for cidr in var.bootstrap_https_ipv6_cidrs : can(cidrnetmask(cidr)) && strcontains(cidr, ":") && cidr != "::/0"])
    error_message = "bootstrap_https_ipv6_cidrs must contain non-default IPv6 CIDRs."
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

locals {
  egress_rules = merge(
    { for index, cidr in sort(tolist(var.cloudflare_tunnel_ipv6_cidrs)) : "cloudflare-tcp-${index}" => { cidr = cidr, protocol = "tcp", from_port = 7844, to_port = 7844 } },
    { for index, cidr in sort(tolist(var.cloudflare_tunnel_ipv6_cidrs)) : "cloudflare-udp-${index}" => { cidr = cidr, protocol = "udp", from_port = 7844, to_port = 7844 } },
    { for index, cidr in sort(tolist(var.management_ipv6_cidrs)) : "ssm-${index}" => { cidr = cidr, protocol = "tcp", from_port = 443, to_port = 443 } },
    { for index, cidr in sort(tolist(var.dns_ipv6_cidrs)) : "dns-tcp-${index}" => { cidr = cidr, protocol = "tcp", from_port = 53, to_port = 53 } },
    { for index, cidr in sort(tolist(var.dns_ipv6_cidrs)) : "dns-udp-${index}" => { cidr = cidr, protocol = "udp", from_port = 53, to_port = 53 } },
    { for index, cidr in sort(tolist(var.bootstrap_https_ipv6_cidrs)) : "bootstrap-${index}" => { cidr = cidr, protocol = "tcp", from_port = 443, to_port = 443 } },
  )
}

resource "aws_vpc" "this" {
  cidr_block                       = var.vpc_ipv4_cidr
  assign_generated_ipv6_cidr_block = true
  enable_dns_hostnames             = true
  enable_dns_support               = true

  tags = merge(var.tags, {
    Name         = "${var.name}-vpc"
    TrialProfile = "aws-ipv6-trial"
  })
}

resource "aws_internet_gateway" "this" {
  vpc_id = aws_vpc.this.id

  tags = merge(var.tags, { Name = "${var.name}-igw" })
}

resource "aws_subnet" "runtime" {
  vpc_id                          = aws_vpc.this.id
  availability_zone               = var.availability_zone
  ipv6_cidr_block                 = cidrsubnet(aws_vpc.this.ipv6_cidr_block, 8, 0)
  ipv6_native                     = true
  assign_ipv6_address_on_creation = true
  enable_dns64                    = false
  map_public_ip_on_launch         = false

  tags = merge(var.tags, {
    Name         = "${var.name}-ipv6-runtime"
    Tier         = "ipv6-only-runtime"
    Architecture = "arm64"
  })
}

resource "aws_route_table" "runtime" {
  vpc_id = aws_vpc.this.id

  route {
    ipv6_cidr_block = "::/0"
    gateway_id      = aws_internet_gateway.this.id
  }

  tags = merge(var.tags, { Name = "${var.name}-ipv6-default" })
}

resource "aws_route_table_association" "runtime" {
  route_table_id = aws_route_table.runtime.id
  subnet_id      = aws_subnet.runtime.id
}

resource "aws_security_group" "runtime" {
  name_prefix            = "${var.name}-runtime-"
  description            = "IPv6-only trial egress; no inbound rules"
  vpc_id                 = aws_vpc.this.id
  revoke_rules_on_delete = true

  # No ingress block or ingress resource is intentional. Cloudflare Tunnel is
  # outbound-only and SSM is reached through outbound dual-stack endpoints.
  tags = merge(var.tags, { Name = "${var.name}-runtime" })
}

resource "aws_vpc_security_group_egress_rule" "runtime" {
  for_each = local.egress_rules

  security_group_id = aws_security_group.runtime.id
  description       = "Reviewed IPv6-only ${each.key} egress"
  cidr_ipv6         = each.value.cidr
  ip_protocol       = each.value.protocol
  from_port         = each.value.from_port
  to_port           = each.value.to_port
}

output "vpc_id" {
  description = "Trial VPC identifier."
  value       = aws_vpc.this.id
}

output "vpc_ipv6_cidr_block" {
  description = "Amazon-provided VPC IPv6 CIDR."
  value       = aws_vpc.this.ipv6_cidr_block
}

output "runtime_subnet_id" {
  description = "The single IPv6-native runtime subnet."
  value       = aws_subnet.runtime.id
}

output "runtime_route_table_id" {
  description = "Route table containing only the IPv6 default route."
  value       = aws_route_table.runtime.id
}

output "runtime_security_group_id" {
  description = "No-ingress runtime security group."
  value       = aws_security_group.runtime.id
}
