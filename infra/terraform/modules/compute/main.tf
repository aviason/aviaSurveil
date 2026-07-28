variable "name" {
  description = "Stable name prefix for compute resources."
  type        = string
}

variable "region" {
  description = "Explicit approved AWS region."
  type        = string
}

variable "private_subnet_ids" {
  description = "Private compute subnet identifiers."
  type        = list(string)

  validation {
    condition     = length(var.private_subnet_ids) >= 2
    error_message = "private_subnet_ids must span at least two subnets."
  }
}

variable "security_group_ids" {
  description = "Private application security group identifiers."
  type        = list(string)
}

variable "instance_profile_name" {
  description = "Resource-scoped runtime instance profile."
  type        = string
}

variable "instance_type" {
  description = "Approved EC2 instance type."
  type        = string
}

variable "ami_id" {
  description = "Approved immutable AMI identifier."
  type        = string

  validation {
    condition     = can(regex("^ami-[0-9a-f]{17}$", var.ami_id))
    error_message = "ami_id must be an explicit long-format AMI identifier."
  }
}

variable "image_uri" {
  description = "Approved private ECR image URI pinned by SHA-256 digest."
  type        = string

  validation {
    condition     = can(regex("@sha256:[0-9a-f]{64}$", var.image_uri))
    error_message = "image_uri must end in an immutable SHA-256 digest."
  }
}

variable "secret_arns" {
  description = "Exact secret references loaded by the runtime."
  type        = list(string)

  validation {
    condition     = length(var.secret_arns) > 0 && alltrue([for arn in var.secret_arns : can(regex("^arn:aws[a-z-]*:secretsmanager:", arn))])
    error_message = "secret_arns must contain explicit Secrets Manager ARNs."
  }
}

variable "kms_key_arn" {
  description = "KMS key for encrypted root EBS volumes."
  type        = string
}

variable "target_group_arns" {
  description = "ALB target groups attached to the autoscaling group."
  type        = list(string)
}

variable "otel_endpoint" {
  description = "Private OpenTelemetry endpoint."
  type        = string
}

variable "min_size" {
  description = "Minimum instance count."
  type        = number
}

variable "desired_capacity" {
  description = "Desired instance count."
  type        = number
}

variable "max_size" {
  description = "Maximum instance count."
  type        = number

  validation {
    condition     = var.max_size > 0
    error_message = "max_size must be positive."
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
    error_message = "tags must include the mandatory ownership and cost keys."
  }
}

locals {
  image_digest = split("@sha256:", var.image_uri)[1]
  user_data    = <<-EOT
    #!/bin/sh
    set -eu
    umask 077
    install -d -m 0700 /run/aviasurveil360/secrets
    for secret_arn in ${join(" ", [for arn in var.secret_arns : "'${arn}'"])}; do
      secret_name="$(printf '%s' "$secret_arn" | shasum -a 256 | awk '{print $1}')"
      aws --region '${var.region}' secretsmanager get-secret-value \
        --secret-id "$secret_arn" \
        --query SecretString \
        --output text >"/run/aviasurveil360/secrets/$secret_name"
      chmod 0600 "/run/aviasurveil360/secrets/$secret_name"
    done
    export OTEL_EXPORTER_OTLP_ENDPOINT='${var.otel_endpoint}'
    image_uri='${var.image_uri}'
    registry_host="$${image_uri%%/*}"
    aws --region '${var.region}' ecr get-login-password |
      docker login --username AWS --password-stdin "$registry_host"
    docker pull "$image_uri"
    docker run --detach --restart unless-stopped \
      --name aviasurveil360 \
      --network host \
      --read-only \
      --tmpfs /tmp:rw,noexec,nosuid,size=128m \
      "$image_uri"
  EOT
}

resource "aws_launch_template" "this" {
  name_prefix            = "${var.name}-"
  description            = "Digest-bound AviaSurveil360 trial runtime"
  image_id               = var.ami_id
  instance_type          = var.instance_type
  update_default_version = true

  iam_instance_profile {
    name = var.instance_profile_name
  }

  metadata_options {
    http_endpoint               = "enabled"
    http_protocol_ipv6          = "disabled"
    http_put_response_hop_limit = 1
    http_tokens                 = "required"
    instance_metadata_tags      = "disabled"
  }

  network_interfaces {
    associate_public_ip_address = false
    delete_on_termination       = true
    device_index                = 0
    security_groups             = var.security_group_ids
  }

  block_device_mappings {
    device_name = "/dev/xvda"

    ebs {
      delete_on_termination = true
      encrypted             = true
      kms_key_id            = var.kms_key_arn
      volume_size           = 30
      volume_type           = "gp3"
    }
  }

  user_data = base64encode(local.user_data)

  tag_specifications {
    resource_type = "instance"
    tags = merge(var.tags, {
      ImageDigest = local.image_digest
      Name        = var.name
    })
  }

  tag_specifications {
    resource_type = "volume"
    tags = merge(var.tags, {
      ImageDigest = local.image_digest
      Name        = "${var.name}-root"
    })
  }

  tags = merge(var.tags, { ImageDigest = local.image_digest })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_autoscaling_group" "this" {
  name_prefix               = "${var.name}-"
  min_size                  = var.min_size
  desired_capacity          = var.desired_capacity
  max_size                  = var.max_size
  vpc_zone_identifier       = var.private_subnet_ids
  target_group_arns         = var.target_group_arns
  health_check_type         = length(var.target_group_arns) > 0 ? "ELB" : "EC2"
  health_check_grace_period = 300
  force_delete              = false

  launch_template {
    id      = aws_launch_template.this.id
    version = "$Latest"
  }

  dynamic "tag" {
    for_each = merge(var.tags, {
      ImageDigest = local.image_digest
      Name        = var.name
    })
    content {
      key                 = tag.key
      value               = tag.value
      propagate_at_launch = true
    }
  }

  instance_refresh {
    strategy = "Rolling"
    preferences {
      min_healthy_percentage = 100
      instance_warmup        = 300
    }
  }

  lifecycle {
    precondition {
      condition     = var.min_size <= var.desired_capacity && var.desired_capacity <= var.max_size
      error_message = "Capacity must satisfy min_size <= desired_capacity <= max_size."
    }
  }
}

output "autoscaling_group_name" {
  description = "Autoscaling group name."
  value       = aws_autoscaling_group.this.name
}

output "launch_template_id" {
  description = "Launch template identifier."
  value       = aws_launch_template.this.id
}
