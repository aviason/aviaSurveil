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

  tags = merge(var.tags, { Purpose = "single-arm64-compose-runtime" })
}

resource "aws_iam_role_policy" "runtime" {
  name = "${var.name}-runtime"
  role = aws_iam_role.runtime.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "InspectConfiguredObjectBuckets"
        Effect   = "Allow"
        Action   = "s3:GetBucketLocation"
        Resource = [for bucket in aws_s3_bucket.objects : bucket.arn]
      },
      {
        Sid    = "UseExactObjectVersions"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:GetObjectVersion",
          "s3:GetObjectTagging",
          "s3:GetObjectVersionTagging",
          "s3:PutObject",
        ]
        Resource = [for bucket in aws_s3_bucket.objects : "${bucket.arn}/*"]
      },
      {
        Sid    = "UseDataAndSecretKeys"
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:DescribeKey",
          "kms:Encrypt",
          "kms:GenerateDataKey",
          "kms:ReEncryptFrom",
          "kms:ReEncryptTo",
        ]
        Resource = [aws_kms_key.data.arn, aws_kms_key.secrets.arn]
      },
      {
        Sid    = "ReadRuntimeSecretReferences"
        Effect = "Allow"
        Action = [
          "secretsmanager:DescribeSecret",
          "secretsmanager:GetSecretValue",
        ]
        Resource = [for secret in aws_secretsmanager_secret.runtime : secret.arn]
      },
      {
        Sid      = "ReadCloudflareConnectorParameter"
        Effect   = "Allow"
        Action   = "ssm:GetParameter"
        Resource = aws_ssm_parameter.cloudflare_connector.arn
      },
      {
        Sid    = "PullExactRuntimeImages"
        Effect = "Allow"
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:BatchGetImage",
          "ecr:GetDownloadUrlForLayer",
        ]
        Resource = [for repository in aws_ecr_repository.runtime : repository.arn]
      },
      {
        Sid      = "ObtainEcrAuthorizationToken"
        Effect   = "Allow"
        Action   = "ecr:GetAuthorizationToken"
        Resource = "*"
      },
      {
        Sid    = "WriteBoundedLogGroups"
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:DescribeLogStreams",
          "logs:PutLogEvents",
        ]
        Resource = [for group in aws_cloudwatch_log_group.runtime : "${group.arn}:*"]
      },
      {
        Sid      = "PublishPrivatePilotMetrics"
        Effect   = "Allow"
        Action   = "cloudwatch:PutMetricData"
        Resource = "*"
        Condition = {
          StringEquals = {
            "cloudwatch:namespace" = "AviaSurveil360/PrivatePilot"
          }
        }
      },
    ]
  })
}

resource "aws_iam_instance_profile" "runtime" {
  name_prefix = "${var.name}-runtime-"
  role        = aws_iam_role.runtime.name
  tags        = var.tags
}

resource "aws_iam_role_policy_attachment" "runtime_ssm" {
  role       = aws_iam_role.runtime.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_s3_bucket_policy" "objects" {
  for_each = aws_s3_bucket.objects

  bucket = each.value.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat(
      [
        {
          Sid       = "DenyInsecureTransport"
          Effect    = "Deny"
          Principal = "*"
          Action    = "s3:*"
          Resource  = [each.value.arn, "${each.value.arn}/*"]
          Condition = {
            Bool = { "aws:SecureTransport" = "false" }
          }
        },
        {
          Sid       = "DenyRuntimeDeletion"
          Effect    = "Deny"
          Principal = { AWS = aws_iam_role.runtime.arn }
          Action    = ["s3:DeleteObject", "s3:DeleteObjectVersion"]
          Resource  = "${each.value.arn}/*"
        },
        {
          Sid       = "DenyRuntimeScanTagMutation"
          Effect    = "Deny"
          Principal = { AWS = aws_iam_role.runtime.arn }
          Action    = ["s3:PutObjectTagging", "s3:PutObjectVersionTagging"]
          Resource  = "${each.value.arn}/*"
        },
      ],
      each.key == "quarantine" ? [
        {
          Sid       = "DenyOrdinaryQuarantineReadUntilExactVersionIsClean"
          Effect    = "Deny"
          Principal = "*"
          Action    = ["s3:GetObject", "s3:GetObjectVersion"]
          Resource  = "${each.value.arn}/*"
          Condition = {
            StringNotEquals = {
              "s3:ExistingObjectTag/GuardDutyMalwareScanStatus" = "NO_THREATS_FOUND"
            }
            ArnNotEquals = {
              "aws:PrincipalArn" = aws_iam_role.guardduty_malware.arn
            }
          }
        },
      ] : [],
    )
  })

  depends_on = [aws_s3_bucket_public_access_block.objects]
}

locals {
  host_bootstrap = <<-EOT
    #!/usr/bin/env bash
    set -euo pipefail
    umask 077

    dnf --setopt=ip_resolve=6 install -y docker docker-compose-plugin amazon-cloudwatch-agent amazon-ecr-credential-helper amazon-ssm-agent curl openssl
    aws --version
    docker compose version
    install -d -m 0755 /opt/aviasurveil360/private-pilot
    install -d -m 0700 /run/aviasurveil360-private-pilot
    install -d -m 0700 /etc/aviasurveil360/private-pilot
    install -d -m 0700 /etc/aviasurveil360/private-pilot/docker

    cat >/etc/aviasurveil360/private-pilot/docker/config.json <<'ECR_CONFIG'
    {
      "credHelpers": {
        "${var.aws_account_id}.dkr-ecr.${var.region}.on.aws": "ecr-login"
      }
    }
    ECR_CONFIG
    chmod 0600 /etc/aviasurveil360/private-pilot/docker/config.json

    cat >/etc/docker/daemon.json <<'DOCKER_CONFIG'
    {
      "live-restore": true,
      "ipv6": true,
      "ip6tables": true,
      "fixed-cidr-v6": "fd36:6176:6961:ffff::/64",
      "log-driver": "local",
      "log-opts": {"max-size": "10m", "max-file": "3"},
      "no-new-privileges": true
    }
    DOCKER_CONFIG

    install -d -m 0755 /etc/amazon/ssm
    cat >/etc/amazon/ssm/amazon-ssm-agent.json <<'SSM_CONFIG'
    {
      "Agent": {
        "Region": "${var.region}",
        "UseDualStackEndpoint": true
      }
    }
    SSM_CONFIG
    chmod 0600 /etc/amazon/ssm/amazon-ssm-agent.json

    install -d -m 0755 /etc/systemd/system/amazon-cloudwatch-agent.service.d
    cat >/etc/systemd/system/amazon-cloudwatch-agent.service.d/dualstack.conf <<'DUALSTACK_CONFIG'
    [Service]
    Environment=AWS_USE_DUALSTACK_ENDPOINT=true
    DUALSTACK_CONFIG

    install -d -m 0755 /etc/systemd/system/amazon-ssm-agent.service.d
    cat >/etc/systemd/system/amazon-ssm-agent.service.d/dualstack.conf <<'DUALSTACK_CONFIG'
    [Service]
    Environment=AWS_USE_DUALSTACK_ENDPOINT=true
    DUALSTACK_CONFIG

    cat >/etc/aviasurveil360/private-pilot/runtime-contract.env <<'RUNTIME_CONTRACT'
    AVIA_RUNTIME_PROFILE=aws-private-pilot
    AVIA_RUNTIME_ARCHITECTURE=linux/arm64
    AVIA_INSTANCE_TYPE=t4g.small
    AVIA_RELEASE_MANIFEST_SHA256=${var.release_manifest_sha256}
    AVIA_AWS_ACCOUNT_ID=${var.aws_account_id}
    AVIA_AWS_REGION=${var.region}
    AVIA_FORCE_IPV6=true
    AVIA_ECR_REGISTRY_HOST=${var.aws_account_id}.dkr-ecr.${var.region}.on.aws
    AVIA_CLOUDFLARE_TUNNEL_TOKEN_PARAMETER_NAME=${var.cloudflare_connector_parameter_name}
    AVIA_CLOUDFLARE_EDGE_IP_VERSION=6
    AVIA_GATEWAY_BIND=127.0.0.1:8080
    RUNTIME_CONTRACT
    chmod 0600 /etc/aviasurveil360/private-pilot/runtime-contract.env

    cat >/opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json <<'CLOUDWATCH_CONFIG'
    {
      "agent": {
        "metrics_collection_interval": 60,
        "run_as_user": "root",
        "use_dualstack_endpoint": true
      },
      "logs": {
        "logs_collected": {
          "files": {
            "collect_list": [
              {
                "file_path": "/var/log/cloud-init-output.log",
                "log_group_name": "/aviasurveil360/${var.name}/host",
                "log_stream_name": "{instance_id}/cloud-init",
                "retention_in_days": -1
              },
              {
                "file_path": "/var/lib/docker/containers/*/*.log",
                "log_group_name": "/aviasurveil360/${var.name}/host",
                "log_stream_name": "{instance_id}/containers",
                "retention_in_days": -1
              }
            ]
          }
        }
      },
      "metrics": {
        "namespace": "AviaSurveil360/PrivatePilot",
        "append_dimensions": {
          "InstanceId": "$${aws:InstanceId}"
        },
        "metrics_collected": {
          "disk": {
            "measurement": ["used_percent"],
            "resources": ["/"],
            "metrics_collection_interval": 60
          },
          "mem": {
            "measurement": ["mem_used_percent"],
            "metrics_collection_interval": 60
          },
          "swap": {
            "measurement": ["swap_used_percent"],
            "metrics_collection_interval": 60
          }
        }
      }
    }
    CLOUDWATCH_CONFIG
    chmod 0600 /opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json

    swapoff --all
    test "$(wc -l </proc/swaps)" -eq 1
    systemctl daemon-reload
    systemctl enable --now docker
    systemctl enable --now amazon-ssm-agent
    /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl \
      -a fetch-config \
      -m ec2 \
      -s \
      -c file:/opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json
    systemctl disable --now sshd.service 2>/dev/null || true

    # Runtime artifacts, image pulls, secrets, Compose start, migrations, and
    # traffic remain separate digest-bound and explicitly authorized actions.
  EOT
}

resource "aws_instance" "runtime" {
  ami                    = var.ami_id
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.dual_stack_app_a.id
  vpc_security_group_ids = [aws_security_group.application.id]
  iam_instance_profile   = aws_iam_instance_profile.runtime.name

  associate_public_ip_address = false
  ipv6_address_count          = 1
  source_dest_check           = true
  user_data                   = local.host_bootstrap
  user_data_replace_on_change = true

  metadata_options {
    http_endpoint      = "enabled"
    http_protocol_ipv6 = "enabled"
    http_tokens        = "required"
    # Docker bridge networking adds one hop. Two permits the approved
    # containers to use IMDSv2 instance-profile credentials without static keys.
    http_put_response_hop_limit = 2
    instance_metadata_tags      = "disabled"
  }

  root_block_device {
    delete_on_termination = false
    encrypted             = true
    kms_key_id            = aws_kms_key.data.arn
    volume_size           = var.root_volume_size_gib
    volume_type           = "gp3"

    tags = merge(var.tags, {
      Name = "${var.name}-runtime-root"
    })
  }

  credit_specification {
    cpu_credits = "standard"
  }

  maintenance_options {
    auto_recovery = "default"
  }

  tags = merge(var.tags, {
    Name                  = "${var.name}-runtime-a"
    Architecture          = "linux-arm64"
    ReleaseManifestSHA256 = var.release_manifest_sha256
    Availability          = "single-az-non-ha"
  })

  lifecycle {
    prevent_destroy = true
    precondition {
      condition     = var.instance_type == "t4g.small"
      error_message = "The private pilot permits exactly one t4g.small runtime."
    }
  }
}
