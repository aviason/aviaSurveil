variable "name" {
  description = "Stable name prefix for network resources."
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block for the trial VPC."
  type        = string
}

variable "availability_zones" {
  description = "Exactly two approved availability zones."
  type        = list(string)

  validation {
    condition     = length(var.availability_zones) == 2 && length(distinct(var.availability_zones)) == 2
    error_message = "availability_zones must contain exactly two distinct zones."
  }
}

variable "public_subnet_cidrs" {
  description = "CIDRs for the ALB subnets."
  type        = list(string)

  validation {
    condition     = length(var.public_subnet_cidrs) == 2
    error_message = "public_subnet_cidrs must contain exactly two CIDRs."
  }
}

variable "compute_subnet_cidrs" {
  description = "CIDRs for private compute subnets."
  type        = list(string)

  validation {
    condition     = length(var.compute_subnet_cidrs) == 2
    error_message = "compute_subnet_cidrs must contain exactly two CIDRs."
  }
}

variable "database_subnet_cidrs" {
  description = "CIDRs for isolated database subnets."
  type        = list(string)

  validation {
    condition     = length(var.database_subnet_cidrs) == 2
    error_message = "database_subnet_cidrs must contain exactly two CIDRs."
  }
}

variable "enable_nat_gateway" {
  description = "Whether private compute subnets receive outbound NAT."
  type        = bool
}

variable "single_nat_gateway" {
  description = "Whether an approved trial accepts one cross-AZ NAT gateway."
  type        = bool

  validation {
    condition     = var.enable_nat_gateway || !var.single_nat_gateway
    error_message = "single_nat_gateway cannot be true when NAT is disabled."
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
  zones = {
    for index, zone in var.availability_zones : tostring(index) => {
      index = index
      zone  = zone
    }
  }
  nat_zones = var.enable_nat_gateway ? (
    var.single_nat_gateway ? { "0" = local.zones["0"] } : local.zones
  ) : {}
}

resource "aws_vpc" "this" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(var.tags, { Name = "${var.name}-vpc" })
}

resource "aws_internet_gateway" "this" {
  vpc_id = aws_vpc.this.id

  tags = merge(var.tags, { Name = "${var.name}-igw" })
}

resource "aws_subnet" "public" {
  for_each = local.zones

  vpc_id                  = aws_vpc.this.id
  availability_zone       = each.value.zone
  cidr_block              = var.public_subnet_cidrs[each.value.index]
  map_public_ip_on_launch = false

  tags = merge(var.tags, {
    Name = "${var.name}-public-${each.value.zone}"
    Tier = "public-alb"
  })
}

resource "aws_subnet" "compute" {
  for_each = local.zones

  vpc_id                  = aws_vpc.this.id
  availability_zone       = each.value.zone
  cidr_block              = var.compute_subnet_cidrs[each.value.index]
  map_public_ip_on_launch = false

  tags = merge(var.tags, {
    Name = "${var.name}-compute-${each.value.zone}"
    Tier = "private-compute"
  })
}

resource "aws_subnet" "database" {
  for_each = local.zones

  vpc_id                  = aws_vpc.this.id
  availability_zone       = each.value.zone
  cidr_block              = var.database_subnet_cidrs[each.value.index]
  map_public_ip_on_launch = false

  tags = merge(var.tags, {
    Name = "${var.name}-database-${each.value.zone}"
    Tier = "isolated-database"
  })
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.this.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.this.id
  }

  tags = merge(var.tags, { Name = "${var.name}-public" })
}

resource "aws_route_table_association" "public" {
  for_each = aws_subnet.public

  route_table_id = aws_route_table.public.id
  subnet_id      = each.value.id
}

resource "aws_eip" "nat" {
  for_each = local.nat_zones

  domain = "vpc"

  tags = merge(var.tags, { Name = "${var.name}-nat-${each.value.zone}" })
}

resource "aws_nat_gateway" "this" {
  for_each = local.nat_zones

  allocation_id = aws_eip.nat[each.key].id
  subnet_id     = aws_subnet.public[each.key].id

  tags = merge(var.tags, { Name = "${var.name}-nat-${each.value.zone}" })

  depends_on = [aws_internet_gateway.this]
}

resource "aws_route_table" "compute" {
  for_each = local.zones

  vpc_id = aws_vpc.this.id

  dynamic "route" {
    for_each = var.enable_nat_gateway ? [true] : []
    content {
      cidr_block = "0.0.0.0/0"
      nat_gateway_id = aws_nat_gateway.this[
        var.single_nat_gateway ? "0" : each.key
      ].id
    }
  }

  tags = merge(var.tags, { Name = "${var.name}-compute-${each.value.zone}" })
}

resource "aws_route_table_association" "compute" {
  for_each = aws_subnet.compute

  route_table_id = aws_route_table.compute[each.key].id
  subnet_id      = each.value.id
}

resource "aws_route_table" "database" {
  for_each = local.zones

  vpc_id = aws_vpc.this.id

  tags = merge(var.tags, { Name = "${var.name}-database-${each.value.zone}" })
}

resource "aws_route_table_association" "database" {
  for_each = aws_subnet.database

  route_table_id = aws_route_table.database[each.key].id
  subnet_id      = each.value.id
}

output "vpc_id" {
  description = "VPC identifier."
  value       = aws_vpc.this.id
}

output "public_subnet_ids" {
  description = "Ordered ALB subnet identifiers."
  value       = [for key in sort(keys(aws_subnet.public)) : aws_subnet.public[key].id]
}

output "private_compute_subnet_ids" {
  description = "Ordered private compute subnet identifiers."
  value       = [for key in sort(keys(aws_subnet.compute)) : aws_subnet.compute[key].id]
}

output "private_database_subnet_ids" {
  description = "Ordered isolated database subnet identifiers."
  value       = [for key in sort(keys(aws_subnet.database)) : aws_subnet.database[key].id]
}

output "private_compute_route_table_ids" {
  description = "Private compute route tables used by gateway endpoints."
  value       = [for key in sort(keys(aws_route_table.compute)) : aws_route_table.compute[key].id]
}
