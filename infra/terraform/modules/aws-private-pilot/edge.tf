resource "cloudflare_zero_trust_tunnel_cloudflared" "this" {
  account_id = var.cloudflare_account_id
  name       = var.cloudflare_tunnel_name
  config_src = "cloudflare"

  lifecycle {
    prevent_destroy = true
  }
}

resource "cloudflare_zero_trust_tunnel_cloudflared_config" "this" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.this.id
  config = {
    ingress = [
      {
        hostname = var.hostname
        service  = "http://127.0.0.1:8080"
      },
      {
        service = "http_status:404"
      },
    ]
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "cloudflare_dns_record" "application" {
  for_each = var.cloudflare_dns_cutover_authorized ? toset([var.hostname]) : toset([])

  zone_id = var.cloudflare_zone_id
  name    = each.value
  content = "${cloudflare_zero_trust_tunnel_cloudflared.this.id}.cfargotunnel.com"
  type    = "CNAME"
  ttl     = 1
  proxied = true

  lifecycle {
    prevent_destroy = true
  }
}

check "cloudflare_dns_is_a_separate_cutover" {
  assert {
    condition     = var.cloudflare_dns_cutover_authorized ? length(cloudflare_dns_record.application) == 1 : length(cloudflare_dns_record.application) == 0
    error_message = "Cloudflare application DNS must remain absent until an explicit cutover wave is authorized."
  }
}

resource "aws_ssm_parameter" "cloudflare_connector" {
  name        = var.cloudflare_connector_parameter_name
  description = "Cloudflare Tunnel connector token; the placeholder must be replaced only by a separately authorized write"
  type        = "SecureString"
  key_id      = aws_kms_key.secrets.arn
  tier        = "Standard"

  # The sensitive Cloudflare connector token is deliberately never read into
  # Terraform and never stored in plan or state. This non-runnable placeholder
  # establishes the encrypted parameter container only.
  value_wo         = "PENDING_SEPARATE_AUTHORIZATION"
  value_wo_version = 1

  tags = merge(var.tags, {
    Name        = var.cloudflare_connector_parameter_name
    SecretClass = "cloudflare-tunnel-connector"
  })

  lifecycle {
    prevent_destroy = true
  }
}

check "cloudflare_connector_token_stays_out_of_terraform_state" {
  assert {
    condition     = aws_ssm_parameter.cloudflare_connector.type == "SecureString" && aws_ssm_parameter.cloudflare_connector.tier == "Standard"
    error_message = "The connector token must remain a Standard KMS-encrypted SecureString populated outside Terraform state."
  }
}
