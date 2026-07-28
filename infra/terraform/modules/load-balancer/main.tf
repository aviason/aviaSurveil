variable "name" {
  description = "Stable name prefix."
  type        = string
}

variable "vpc_id" {
  description = "VPC identifier."
  type        = string
}

variable "public_subnet_ids" {
  description = "Two public ALB subnet identifiers."
  type        = list(string)

  validation {
    condition     = length(var.public_subnet_ids) == 2
    error_message = "public_subnet_ids must contain exactly two subnets."
  }
}

variable "security_group_id" {
  description = "HTTPS-only ALB security group."
  type        = string
}

variable "certificate_arn" {
  description = "Approved ACM certificate ARN."
  type        = string
}

variable "target_port" {
  description = "Private application target port."
  type        = number
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

resource "aws_lb" "this" {
  name = substr(var.name, 0, 32)
  #trivy:ignore:AVD-AWS-0053 The trial's only public entrypoint is the protected HTTPS listener.
  internal                   = false
  load_balancer_type         = "application"
  security_groups            = [var.security_group_id]
  subnets                    = var.public_subnet_ids
  drop_invalid_header_fields = true
  enable_deletion_protection = true

  tags = var.tags
}

resource "aws_lb_target_group" "application" {
  name        = substr("${var.name}-app", 0, 32)
  port        = var.target_port
  protocol    = "HTTPS"
  target_type = "instance"
  vpc_id      = var.vpc_id

  health_check {
    enabled             = true
    healthy_threshold   = 2
    interval            = 30
    matcher             = "200"
    path                = "/health/ready"
    port                = "traffic-port"
    protocol            = "HTTPS"
    timeout             = 5
    unhealthy_threshold = 3
  }

  deregistration_delay = 30

  tags = var.tags
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.this.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = var.certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.application.arn
  }

  tags = var.tags
}

output "load_balancer_arn" {
  description = "ALB ARN."
  value       = aws_lb.this.arn
}

output "dns_name" {
  description = "ALB DNS name."
  value       = aws_lb.this.dns_name
}

output "target_group_arn" {
  description = "Application target group ARN."
  value       = aws_lb_target_group.application.arn
}
