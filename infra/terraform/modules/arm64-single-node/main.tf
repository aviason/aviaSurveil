variable "name" {
  description = "Stable name prefix for the one-node ARM64 runtime."
  type        = string
}

variable "region" {
  description = "Approved AWS region used by bootstrap and image pulls."
  type        = string
}

variable "subnet_id" {
  description = "The single IPv6-native subnet identifier."
  type        = string
}

variable "security_group_id" {
  description = "No-ingress runtime security group."
  type        = string
}

variable "ami_ssm_parameter_name" {
  description = "AWS-owned AL2023 ARM64 public SSM parameter."
  type        = string

  validation {
    condition     = can(regex("^/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64$", var.ami_ssm_parameter_name))
    error_message = "ami_ssm_parameter_name must be the AWS-owned AL2023 ARM64 parameter."
  }
}

variable "ami_id" {
  description = "Optional owner-approved resolved AMI ID; empty means resolve the SSM parameter during an authorized plan."
  type        = string
  default     = ""

  validation {
    condition     = var.ami_id == "" || can(regex("^ami-[0-9a-f]{17}$", var.ami_id))
    error_message = "ami_id must be empty or an explicit long-format AMI identifier."
  }
}

variable "instance_type" {
  description = "The only supported trial instance type."
  type        = string

  validation {
    condition     = var.instance_type == "t4g.small"
    error_message = "The IPv6 trial is fixed to t4g.small."
  }
}

variable "root_volume_size_gib" {
  description = "Bounded encrypted gp3 root volume size."
  type        = number

  validation {
    condition     = var.root_volume_size_gib == floor(var.root_volume_size_gib) && var.root_volume_size_gib >= 8 && var.root_volume_size_gib <= 64
    error_message = "root_volume_size_gib must be an integer between 8 and 64 GiB."
  }
}

variable "delete_root_volume_on_termination" {
  description = "Owner decision for disposable root-volume deletion."
  type        = bool
}

variable "ecr_repository_arns" {
  description = "Exact private ECR repositories used by the milestone runtime."
  type        = set(string)

  validation {
    condition     = length(var.ecr_repository_arns) == 2 && alltrue([for arn in var.ecr_repository_arns : can(regex("^arn:aws[a-z-]*:ecr:[^:]+:\\d{12}:repository/[^*]+$", arn))])
    error_message = "ecr_repository_arns must contain exactly two explicit repository ARNs."
  }
}

variable "image_uris" {
  description = "Gateway and web-demo ECR subjects pinned by digest."
  type        = map(string)

  validation {
    condition     = length(var.image_uris) == 2 && setsubtract(["gateway", "web-demo"], keys(var.image_uris)) == toset([]) && alltrue([for image in values(var.image_uris) : can(regex("^[0-9]{12}\\.dkr-ecr\\.[a-z0-9-]+\\.on\\.aws/.+@sha256:[0-9a-f]{64}$", image))])
    error_message = "image_uris must contain exactly digest-bound private ECR .on.aws gateway and web-demo subjects."
  }
}

variable "cloudflared_image" {
  description = "Digest-bound linux/arm64 cloudflared image subject."
  type        = string

  validation {
    condition     = can(regex("^.+@sha256:[0-9a-f]{64}$", var.cloudflared_image))
    error_message = "cloudflared_image must be an immutable digest-bound subject."
  }
}

variable "ecr_registry_host" {
  description = "IPv6-capable ECR registry host ending in .on.aws."
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}\\.dkr-ecr\\.[a-z0-9-]+\\.on\\.aws$", var.ecr_registry_host))
    error_message = "ecr_registry_host must use the IPv6-capable .dkr-ecr.<region>.on.aws endpoint."
  }
}

variable "cloudflare_connector_parameter_arn" {
  description = "Exact SSM SecureString parameter ARN containing the tunnel token."
  type        = string

  validation {
    condition     = can(regex("^arn:aws[a-z-]*:ssm:[^:]+:\\d{12}:parameter/[^*]+$", var.cloudflare_connector_parameter_arn))
    error_message = "cloudflare_connector_parameter_arn must be an explicit SSM parameter ARN."
  }
}

variable "compose_bundle_path" {
  description = "Owner-approved path for the immutable Compose bundle installed by a later artifact action."
  type        = string
}

variable "compose_bundle_sha256" {
  description = "Digest of the Compose bundle bound to the runtime."
  type        = string

  validation {
    condition     = can(regex("^[0-9a-f]{64}$", var.compose_bundle_sha256))
    error_message = "compose_bundle_sha256 must be a 64-character lowercase SHA-256 digest."
  }
}

variable "kms_key_arn" {
  description = "KMS key used for the encrypted root volume and runtime parameter access."
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
    error_message = "tags must include non-empty Environment, Owner, CostCenter, DataClassification, and ManagedBy values."
  }
}

data "aws_ssm_parameter" "arm64_ami" {
  count = var.ami_id == "" ? 1 : 0
  name  = var.ami_ssm_parameter_name
}

locals {
  resolved_ami_id = var.ami_id != "" ? var.ami_id : data.aws_ssm_parameter.arm64_ami[0].value
  image_digest_tag = join(",", [
    for key in ["gateway", "web-demo"] : split("@sha256:", var.image_uris[key])[1]
  ])
  user_data = <<-EOT
    #!/usr/bin/env bash
    set -euo pipefail
    umask 077

    install -d -m 0700 /etc/aviasurveil360 /run/aviasurveil360/secrets
    dnf install -y docker amazon-ssm-agent
    install -d -m 0755 /etc/amazon/ssm
    cat >/etc/amazon/ssm/amazon-ssm-agent.json <<'SSM_CONFIG'
    {"UseDualStackEndpoint":true}
    SSM_CONFIG

    systemctl enable --now docker
    systemctl enable --now amazon-ssm-agent
    systemctl disable --now sshd.service 2>/dev/null || true
    export AWS_USE_DUALSTACK_ENDPOINT=true

    cat >/etc/aviasurveil360/trial.env <<'RUNTIME_CONFIG'
    AVIA_TRIAL_PROFILE=demo
    AVIA_TRIAL_ARCHITECTURE=linux/arm64
    AVIA_TRIAL_INSTANCE_TYPE=t4g.small
    AVIA_TRIAL_COMPOSE_BUNDLE_PATH=${var.compose_bundle_path}
    AVIA_TRIAL_COMPOSE_BUNDLE_SHA256=${var.compose_bundle_sha256}
    AVIA_TRIAL_CLOUDFLARE_TOKEN_PARAMETER_ARN=${var.cloudflare_connector_parameter_arn}
    AVIA_TRIAL_ECR_REGISTRY_HOST=${var.ecr_registry_host}
    AVIA_TRIAL_CLOUDFLARED_IMAGE=${var.cloudflared_image}
    AVIA_TRIAL_GATEWAY_IMAGE=${var.image_uris["gateway"]}
    AVIA_TRIAL_WEB_DEMO_IMAGE=${var.image_uris["web-demo"]}
    TUNNEL_EDGE_IP_VERSION=6
    RUNTIME_CONFIG
    chmod 0600 /etc/aviasurveil360/trial.env

    aws --region '${var.region}' ssm get-parameter \
      --name '${var.cloudflare_connector_parameter_arn}' \
      --with-decryption \
      --query Parameter.Value \
      --output text >/run/aviasurveil360/secrets/cloudflare-tunnel-token
    chmod 0600 /run/aviasurveil360/secrets/cloudflare-tunnel-token

    aws --region '${var.region}' ecr get-login-password \
      --endpoint-url "https://ecr.${var.region}.api.aws" |
      docker login --username AWS --password-stdin '${var.ecr_registry_host}'
    docker pull '${var.cloudflared_image}'
    docker pull '${var.image_uris["gateway"]}'
    docker pull '${var.image_uris["web-demo"]}'
  EOT
}

resource "aws_iam_role" "runtime" {
  name_prefix = "${var.name}-runtime-"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
  tags = merge(var.tags, { Name = "${var.name}-runtime-role" })
}

resource "aws_iam_role_policy" "runtime" {
  name = "${var.name}-runtime-policy"
  role = aws_iam_role.runtime.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ReadConnectorParameter"
        Effect   = "Allow"
        Action   = ["ssm:GetParameter"]
        Resource = var.cloudflare_connector_parameter_arn
      },
      {
        Sid      = "DecryptConnectorParameter"
        Effect   = "Allow"
        Action   = ["kms:Decrypt", "kms:DescribeKey"]
        Resource = var.kms_key_arn
      },
      {
        Sid      = "PullMilestoneImages"
        Effect   = "Allow"
        Action   = ["ecr:BatchCheckLayerAvailability", "ecr:BatchGetImage", "ecr:GetDownloadUrlForLayer"]
        Resource = tolist(var.ecr_repository_arns)
      },
      {
        Sid      = "EcrLogin"
        Effect   = "Allow"
        Action   = ["ecr:GetAuthorizationToken"]
        Resource = "*"
      },
      {
        Sid      = "SessionManagerControlPlane"
        Effect   = "Allow"
        Action   = ["ssm:DescribeAssociation", "ssm:GetDeployablePatchSnapshotForInstance", "ssm:GetDocument", "ssm:GetManifest", "ssm:ListAssociations", "ssm:PutInventory", "ssm:PutComplianceItems", "ssm:PutConfigurePackageResult", "ssm:UpdateAssociationStatus", "ssm:UpdateInstanceInformation", "ssmmessages:CreateControlChannel", "ssmmessages:CreateDataChannel", "ssmmessages:OpenControlChannel", "ssmmessages:OpenDataChannel", "ec2messages:AcknowledgeMessage", "ec2messages:DeleteMessage", "ec2messages:FailMessage", "ec2messages:GetEndpoint", "ec2messages:GetMessages", "ec2messages:SendReply"]
        Resource = "*"
      },
    ]
  })
}

resource "aws_iam_instance_profile" "runtime" {
  name_prefix = "${var.name}-runtime-"
  role        = aws_iam_role.runtime.name
  tags        = merge(var.tags, { Name = "${var.name}-runtime-profile" })
}

resource "aws_instance" "runtime" {
  ami                         = local.resolved_ami_id
  instance_type               = var.instance_type
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = [var.security_group_id]
  ipv6_address_count          = 1
  associate_public_ip_address = false
  iam_instance_profile        = aws_iam_instance_profile.runtime.name
  user_data                   = base64encode(local.user_data)
  user_data_replace_on_change = true
  monitoring                  = false
  source_dest_check           = true

  metadata_options {
    http_endpoint               = "enabled"
    http_protocol_ipv6          = "enabled"
    http_put_response_hop_limit = 1
    http_tokens                 = "required"
    instance_metadata_tags      = "disabled"
  }

  root_block_device {
    device_name           = "/dev/xvda"
    encrypted             = true
    kms_key_id            = var.kms_key_arn
    volume_type           = "gp3"
    volume_size           = var.root_volume_size_gib
    delete_on_termination = var.delete_root_volume_on_termination
  }

  tags = merge(var.tags, {
    Name         = var.name
    TrialProfile = "aws-ipv6-trial"
    Architecture = "arm64"
    InstanceType = "t4g.small"
    ImageDigests = local.image_digest_tag
  })
}

output "instance_id" {
  description = "The single ARM64 trial instance."
  value       = aws_instance.runtime.id
}

output "instance_ipv6_address" {
  description = "Primary IPv6 address; there is no public IPv4 output."
  value       = aws_instance.runtime.ipv6_addresses[0]
}

output "runtime_role_arn" {
  description = "Instance role used for SSM, ECR, and the connector parameter."
  value       = aws_iam_role.runtime.arn
}

output "instance_profile_name" {
  description = "Instance profile attached to the node."
  value       = aws_iam_instance_profile.runtime.name
}
