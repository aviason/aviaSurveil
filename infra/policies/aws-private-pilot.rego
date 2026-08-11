package aviasurveil360.aws_private_pilot

import rego.v1

forbidden_types := {
  "aws_autoscaling_group",
  "aws_launch_template",
  "aws_rds_cluster",
  "aws_iam_access_key",
  "aws_secretsmanager_secret_version",
  "aws_wafv2_web_acl",
  "aws_ses_domain_identity",
  "aws_ses_email_identity",
  "aws_internet_gateway",
  "aws_eip",
  "aws_lb",
  "aws_lb_listener",
  "aws_lb_listener_rule",
  "aws_lb_target_group",
  "aws_lb_target_group_attachment",
}

managed(resource) if {
  object.get(resource, "mode", "managed") == "managed"
  object.get(object.get(resource, "change", {}), "actions", []) != ["delete"]
}

after(resource) := object.get(object.get(resource, "change", {}), "after", {})

resources_of_type(resource_type) := [resource |
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == resource_type
]

is_wildcard(value) if value == "*"

is_wildcard(value) if {
  is_array(value)
  some item in value
  item == "*"
}

statement_actions(statement) := [object.get(statement, "Action", "")] if is_string(object.get(statement, "Action", ""))

statement_actions(statement) := object.get(statement, "Action", []) if is_array(object.get(statement, "Action", []))

policy_statements(policy) := policy.Statement if is_array(object.get(policy, "Statement", []))

policy_statements(policy) := [policy.Statement] if is_object(object.get(policy, "Statement", {}))

allowed_wildcard_resource(statement) if {
  object.get(statement, "Sid", "") == "ObtainEcrAuthorizationToken"
  statement_actions(statement) == ["ecr:GetAuthorizationToken"]
}

allowed_wildcard_resource(statement) if {
  object.get(statement, "Sid", "") == "PublishPrivatePilotMetrics"
  statement_actions(statement) == ["cloudwatch:PutMetricData"]
  object.get(object.get(statement, "Condition", {}), "StringEquals", {})["cloudwatch:namespace"] == "AviaSurveil360/PrivatePilot"
}

deny contains sprintf("%s creates a forbidden private-pilot resource type %s", [resource.address, resource.type]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type in forbidden_types
}

deny contains "the private pilot must contain exactly one EC2 instance" if {
  count(resources_of_type("aws_instance")) != 1
}

deny contains "the private pilot must not contain a NAT Gateway" if {
  count(resources_of_type("aws_nat_gateway")) != 0
}

deny contains "the private pilot must contain exactly one egress-only Internet Gateway" if {
  count(resources_of_type("aws_egress_only_internet_gateway")) != 1
}

deny contains "the private pilot must contain exactly one RDS instance" if {
  count(resources_of_type("aws_db_instance")) != 1
}

deny contains "the private pilot must contain exactly one S3 Gateway Endpoint" if {
  endpoints := resources_of_type("aws_vpc_endpoint")
  count(endpoints) != 1
}

deny contains sprintf("%s must be the one private t4g.small", [resource.address]) if {
  some resource in resources_of_type("aws_instance")
  object.get(after(resource), "instance_type", "") != "t4g.small"
}

deny contains sprintf("%s must have exactly one global IPv6 address", [resource.address]) if {
  some resource in resources_of_type("aws_instance")
  object.get(after(resource), "ipv6_address_count", 0) != 1
}

deny contains sprintf("%s must be the one private t4g.small", [resource.address]) if {
  some resource in resources_of_type("aws_instance")
  object.get(after(resource), "associate_public_ip_address", true) != false
}

deny contains sprintf("%s must require IMDSv2", [resource.address]) if {
  some resource in resources_of_type("aws_instance")
  metadata := object.get(after(resource), "metadata_options", [{}])[0]
  object.get(metadata, "http_tokens", "") != "required"
}

deny contains sprintf("%s must use encrypted gp3 root storage", [resource.address]) if {
  some resource in resources_of_type("aws_instance")
  root := object.get(after(resource), "root_block_device", [{}])[0]
  object.get(root, "encrypted", false) != true or object.get(root, "volume_type", "") != "gp3"
}

deny contains sprintf("%s must remain a private encrypted Single-AZ db.t4g.micro", [resource.address]) if {
  some resource in resources_of_type("aws_db_instance")
  database := after(resource)
  object.get(database, "instance_class", "") != "db.t4g.micro" or
    object.get(database, "multi_az", true) != false or
    object.get(database, "publicly_accessible", true) != false or
    object.get(database, "storage_encrypted", false) != true
}

deny contains sprintf("%s must keep 14-day PITR and destructive guards", [resource.address]) if {
  some resource in resources_of_type("aws_db_instance")
  database := after(resource)
  object.get(database, "backup_retention_period", 0) != 14 or
    object.get(database, "deletion_protection", false) != true or
    object.get(database, "skip_final_snapshot", true) != false
}

deny contains sprintf("%s creates interface-endpoint sprawl", [resource.address]) if {
  some resource in resources_of_type("aws_vpc_endpoint")
  object.get(after(resource), "vpc_endpoint_type", "") != "Gateway"
}

deny contains sprintf("%s permits public S3 access", [resource.address]) if {
  some resource in resources_of_type("aws_s3_bucket_public_access_block")
  some key in {"block_public_acls", "block_public_policy", "ignore_public_acls", "restrict_public_buckets"}
  object.get(after(resource), key, false) != true
}

deny contains sprintf("%s disables S3 versioning", [resource.address]) if {
  some resource in resources_of_type("aws_s3_bucket_versioning")
  configuration := object.get(after(resource), "versioning_configuration", [{}])[0]
  object.get(configuration, "status", "") != "Enabled"
}

deny contains sprintf("%s must use standalone GuardDuty result tagging", [resource.address]) if {
  some resource in resources_of_type("aws_guardduty_malware_protection_plan")
  actions := object.get(after(resource), "actions", [{}])[0]
  tagging := object.get(actions, "tagging", [{}])[0]
  object.get(tagging, "status", "") != "ENABLED"
}

deny contains sprintf("%s must keep immutable scanned ECR subjects", [resource.address]) if {
  some resource in resources_of_type("aws_ecr_repository")
  repository := after(resource)
  scan := object.get(repository, "image_scanning_configuration", [{}])[0]
  object.get(repository, "image_tag_mutability", "") != "IMMUTABLE" or object.get(scan, "scan_on_push", false) != true
}

deny contains sprintf("%s must remain Cloudflare-proxied", [resource.address]) if {
  some resource in resources_of_type("cloudflare_dns_record")
  object.get(after(resource), "proxied", false) != true
}

deny contains sprintf("%s must remain a CNAME", [resource.address]) if {
  some resource in resources_of_type("cloudflare_dns_record")
  object.get(after(resource), "type", "") != "CNAME"
}

deny contains "the private pilot may manage at most one separately authorized Cloudflare DNS cutover record" if {
  count(resources_of_type("cloudflare_dns_record")) > 1
}

deny contains "the private pilot must include exactly one remotely managed cloudflare_zero_trust_tunnel_cloudflared" if {
  count(resources_of_type("cloudflare_zero_trust_tunnel_cloudflared")) != 1
}

deny contains "the private pilot must include exactly one Cloudflare Tunnel ingress configuration" if {
  count(resources_of_type("cloudflare_zero_trust_tunnel_cloudflared_config")) != 1
}

deny contains "the private pilot must include exactly one KMS-encrypted connector parameter" if {
  parameters := resources_of_type("aws_ssm_parameter")
  count(parameters) != 1
}

deny contains sprintf("%s must remain a Standard SecureString connector container", [resource.address]) if {
  some resource in resources_of_type("aws_ssm_parameter")
  parameter := after(resource)
  object.get(parameter, "type", "") != "SecureString" or object.get(parameter, "tier", "") != "Standard"
}

deny contains "the private pilot must include an AWS Budget" if {
  count(resources_of_type("aws_budgets_budget")) != 1
}

deny contains sprintf("%s contains wildcard IAM action authority", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type in {"aws_iam_policy", "aws_iam_role_policy"}
  policy := json.unmarshal(object.get(after(resource), "policy", "{}"))
  some statement in policy_statements(policy)
  is_wildcard(object.get(statement, "Action", []))
}

deny contains sprintf("%s contains broad wildcard IAM resource authority", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type in {"aws_iam_policy", "aws_iam_role_policy"}
  policy := json.unmarshal(object.get(after(resource), "policy", "{}"))
  some statement in policy_statements(policy)
  is_wildcard(object.get(statement, "Resource", []))
  not allowed_wildcard_resource(statement)
}
