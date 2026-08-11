locals {
  zones = {
    a = {
      index = 0
      name  = var.availability_zones[0]
    }
    b = {
      index = 1
      name  = var.availability_zones[1]
    }
  }
  cloudflare_tunnel_egress_rules = merge(
    { for index, cidr in sort(tolist(var.cloudflare_tunnel_ipv6_cidrs)) : "tcp-${index}" => { cidr = cidr, protocol = "tcp" } },
    { for index, cidr in sort(tolist(var.cloudflare_tunnel_ipv6_cidrs)) : "udp-${index}" => { cidr = cidr, protocol = "udp" } },
  )
}

resource "aws_vpc" "this" {
  cidr_block                       = var.vpc_cidr
  assign_generated_ipv6_cidr_block = true
  enable_dns_hostnames             = true
  enable_dns_support               = true

  tags = merge(var.tags, {
    Name         = "${var.name}-vpc"
    PilotShape   = "single-workload-az-dual-stack"
    Architecture = "linux-arm64"
  })
}

resource "aws_iam_role" "vpc_flow_logs" {
  name_prefix = "${var.name}-vpc-flow-"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "vpc-flow-logs.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
  tags = var.tags
}

resource "aws_iam_role_policy" "vpc_flow_logs" {
  name = "${var.name}-vpc-flow"
  role = aws_iam_role.vpc_flow_logs.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "logs:CreateLogStream",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams",
        "logs:PutLogEvents",
      ]
      Resource = "${aws_cloudwatch_log_group.runtime["vpc-flow"].arn}:*"
    }]
  })
}

resource "aws_flow_log" "vpc" {
  iam_role_arn    = aws_iam_role.vpc_flow_logs.arn
  log_destination = aws_cloudwatch_log_group.runtime["vpc-flow"].arn
  traffic_type    = "ALL"
  vpc_id          = aws_vpc.this.id

  max_aggregation_interval = 60
  tags                     = merge(var.tags, { Name = "${var.name}-vpc-flow" })
}

resource "aws_egress_only_internet_gateway" "this" {
  vpc_id = aws_vpc.this.id
  tags   = merge(var.tags, { Name = "${var.name}-ipv6-egress-only" })
}

resource "aws_subnet" "dual_stack_app_a" {
  vpc_id                          = aws_vpc.this.id
  availability_zone               = var.availability_zones[0]
  cidr_block                      = var.dual_stack_app_subnet_cidr
  ipv6_cidr_block                 = cidrsubnet(aws_vpc.this.ipv6_cidr_block, 8, 0)
  assign_ipv6_address_on_creation = true
  enable_dns64                    = false
  map_public_ip_on_launch         = false

  tags = merge(var.tags, {
    Name         = "${var.name}-dual-stack-app-a"
    Tier         = "private-dual-stack-application"
    WorkloadZone = "a"
  })
}

resource "aws_subnet" "private_database" {
  for_each = local.zones

  vpc_id                  = aws_vpc.this.id
  availability_zone       = each.value.name
  cidr_block              = var.private_database_subnet_cidrs[each.value.index]
  map_public_ip_on_launch = false

  tags = merge(var.tags, {
    Name = "${var.name}-private-db-${each.key}"
    Tier = "private-ipv4-database"
  })
}

resource "aws_route_table" "dual_stack_app_a" {
  vpc_id = aws_vpc.this.id

  route {
    ipv6_cidr_block        = "::/0"
    egress_only_gateway_id = aws_egress_only_internet_gateway.this.id
  }

  tags = merge(var.tags, { Name = "${var.name}-dual-stack-app-a" })
}

resource "aws_route_table_association" "dual_stack_app_a" {
  route_table_id = aws_route_table.dual_stack_app_a.id
  subnet_id      = aws_subnet.dual_stack_app_a.id
}

resource "aws_route_table" "private_database" {
  for_each = local.zones

  vpc_id = aws_vpc.this.id
  tags   = merge(var.tags, { Name = "${var.name}-private-db-${each.key}" })
}

resource "aws_route_table_association" "private_database" {
  for_each = aws_subnet.private_database

  route_table_id = aws_route_table.private_database[each.key].id
  subnet_id      = each.value.id
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id            = aws_vpc.this.id
  service_name      = "com.amazonaws.${var.region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = [aws_route_table.dual_stack_app_a.id]

  tags = merge(var.tags, { Name = "${var.name}-s3-gateway" })
}

resource "aws_security_group" "application" {
  name_prefix            = "${var.name}-application-"
  description            = "The sole dual-stack ARM64 host; intentionally no ingress rules"
  vpc_id                 = aws_vpc.this.id
  revoke_rules_on_delete = true

  tags = merge(var.tags, { Name = "${var.name}-application-no-ingress" })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "database" {
  name_prefix            = "${var.name}-database-"
  description            = "Single-AZ RDS access from the one application host"
  vpc_id                 = aws_vpc.this.id
  revoke_rules_on_delete = true

  tags = merge(var.tags, { Name = "${var.name}-database" })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_egress_rule" "application_database" {
  security_group_id            = aws_security_group.application.id
  description                  = "Application and Keycloak to the shared private RDS instance"
  referenced_security_group_id = aws_security_group.database.id
  from_port                    = 5432
  ip_protocol                  = "tcp"
  to_port                      = 5432
}

resource "aws_vpc_security_group_ingress_rule" "database_application" {
  security_group_id            = aws_security_group.database.id
  description                  = "PostgreSQL from the sole no-ingress application host"
  referenced_security_group_id = aws_security_group.application.id
  from_port                    = 5432
  ip_protocol                  = "tcp"
  to_port                      = 5432
}

resource "aws_vpc_security_group_egress_rule" "application_s3" {
  security_group_id = aws_security_group.application.id
  description       = "TLS to private S3 through the IPv4 Gateway Endpoint"
  prefix_list_id    = aws_vpc_endpoint.s3.prefix_list_id
  from_port         = 443
  ip_protocol       = "tcp"
  to_port           = 443
}

resource "aws_vpc_security_group_egress_rule" "application_https_ipv6" {
  security_group_id = aws_security_group.application.id
  description       = "Certificate-verified IPv6 HTTPS for AWS dual-stack APIs, ECR, bootstrap repositories, and approved upstreams"
  #trivy:ignore:AVD-AWS-0104 This supersedes the accepted IPv4 0.0.0.0/0:443 NAT path; there is no IPv4 default route and application protocols still fail closed on certificate verification.
  cidr_ipv6   = "::/0"
  from_port   = 443
  ip_protocol = "tcp"
  to_port     = 443
}

resource "aws_vpc_security_group_egress_rule" "application_cloudflare_tunnel" {
  for_each = local.cloudflare_tunnel_egress_rules

  security_group_id = aws_security_group.application.id
  description       = "Reviewed Cloudflare Tunnel IPv6 ${each.value.protocol} edge range"
  cidr_ipv6         = each.value.cidr
  from_port         = 7844
  ip_protocol       = each.value.protocol
  to_port           = 7844
}

resource "aws_vpc_security_group_egress_rule" "application_smtp" {
  for_each = var.smtp_ipv6_cidrs

  security_group_id = aws_security_group.application.id
  description       = "Owner-approved certificate-verified encrypted SMTP IPv6 egress"
  cidr_ipv6         = each.value
  from_port         = var.smtp_port
  ip_protocol       = "tcp"
  to_port           = var.smtp_port
}

resource "aws_vpc_security_group_egress_rule" "application_dns_udp" {
  security_group_id = aws_security_group.application.id
  description       = "Amazon-provided VPC resolver DNS over UDP"
  cidr_ipv4         = "${cidrhost(var.vpc_cidr, 2)}/32"
  from_port         = 53
  ip_protocol       = "udp"
  to_port           = 53
}

resource "aws_vpc_security_group_egress_rule" "application_dns_tcp" {
  security_group_id = aws_security_group.application.id
  description       = "Amazon-provided VPC resolver DNS over TCP"
  cidr_ipv4         = "${cidrhost(var.vpc_cidr, 2)}/32"
  from_port         = 53
  ip_protocol       = "tcp"
  to_port           = 53
}
