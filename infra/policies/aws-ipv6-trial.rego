package aviasurveil360.aws_ipv6_trial

import rego.v1

required_tags := {
  "Environment",
  "Owner",
  "CostCenter",
  "DataClassification",
  "ManagedBy",
}

forbidden_types := {
  "aws_autoscaling_group",
  "aws_launch_template",
  "aws_eip",
  "aws_lb",
  "aws_lb_listener",
  "aws_lb_target_group",
  "aws_nat_gateway",
  "aws_vpc_endpoint",
  "aws_db_instance",
  "aws_db_subnet_group",
  "aws_rds_cluster",
}

managed(resource) if {
  object.get(resource, "mode", "managed") == "managed"
  object.get(object.get(resource, "change", {}), "actions", []) != ["delete"]
}

after(resource) := object.get(object.get(resource, "change", {}), "after", {})

tags(resource) := object.get(after(resource), "tags", {})

is_wildcard(value) if value == "*"

is_wildcard(value) if {
  is_array(value)
  some item in value
  item == "*"
}

required_control_plane_wildcard_actions := {
  "ecr:GetAuthorizationToken",
  "ssm:DescribeAssociation",
  "ssm:GetDeployablePatchSnapshotForInstance",
  "ssm:GetDocument",
  "ssm:GetManifest",
  "ssm:ListAssociations",
  "ssm:PutInventory",
  "ssm:PutComplianceItems",
  "ssm:PutConfigurePackageResult",
  "ssm:UpdateInstanceInformation",
  "ssm:UpdateAssociationStatus",
  "ssmmessages:CreateControlChannel",
  "ssmmessages:CreateDataChannel",
  "ssmmessages:OpenControlChannel",
  "ssmmessages:OpenDataChannel",
  "ec2messages:AcknowledgeMessage",
  "ec2messages:DeleteMessage",
  "ec2messages:FailMessage",
  "ec2messages:GetEndpoint",
  "ec2messages:GetMessages",
  "ec2messages:SendReply",
}

statement_actions(statement) := [object.get(statement, "Action", "")] if is_string(object.get(statement, "Action", ""))

statement_actions(statement) := object.get(statement, "Action", []) if is_array(object.get(statement, "Action", []))

wildcard_resource_is_allowed(statement) if {
  actions := statement_actions(statement)
  count(actions) > 0
  every action in actions {
    action in required_control_plane_wildcard_actions
  }
}

policy_statements(policy) := policy.Statement if is_array(object.get(policy, "Statement", []))

policy_statements(policy) := [policy.Statement] if is_object(object.get(policy, "Statement", {}))

deny contains sprintf("%s creates a prohibited paid or multi-node resource type %s", [resource.address, resource.type]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type in forbidden_types
}

deny contains sprintf("%s is missing mandatory tag %s", [resource.address, tag]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type in {
    "aws_instance",
    "aws_budgets_budget",
    "aws_iam_instance_profile",
    "aws_iam_role",
    "aws_internet_gateway",
    "aws_route_table",
    "aws_security_group",
    "aws_subnet",
    "aws_vpc",
    "aws_ssm_parameter",
  }
  some tag in required_tags
  object.get(tags(resource), tag, "") == ""
}

deny contains sprintf("%s must be exactly one ARM64 t4g.small instance", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_instance"
  object.get(after(resource), "instance_type", "") != "t4g.small"
}

deny contains sprintf("%s must not receive a public IPv4 address", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_instance"
  object.get(after(resource), "associate_public_ip_address", true) != false
}

deny contains sprintf("%s must have a primary IPv6 address", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_instance"
  object.get(after(resource), "ipv6_address_count", 0) < 1
}

deny contains sprintf("%s must have exactly one IPv6 address", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_instance"
  object.get(after(resource), "ipv6_address_count", 0) != 1
}

deny contains sprintf("%s must require IMDSv2 and enable IMDS IPv6", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_instance"
  options := object.get(after(resource), "metadata_options", [{}])[0]
  object.get(options, "http_tokens", "") != "required"
}

deny contains sprintf("%s must require IMDSv2 and enable IMDS IPv6", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_instance"
  options := object.get(after(resource), "metadata_options", [{}])[0]
  object.get(options, "http_protocol_ipv6", "") != "enabled"
}

deny contains sprintf("%s must not attach an SSH key", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_instance"
  object.get(after(resource), "key_name", null) != null
}

deny contains sprintf("%s must use an IPv6-native subnet with no public IPv4 mapping", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_subnet"
  object.get(after(resource), "ipv6_native", false) != true
}

deny contains sprintf("%s must use an IPv6-native subnet with no public IPv4 mapping", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_subnet"
  object.get(after(resource), "map_public_ip_on_launch", true) != false
}

deny contains sprintf("%s must use an IPv6-native subnet with no IPv4 CIDR", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_subnet"
  object.get(after(resource), "cidr_block", "") != ""
}

deny contains sprintf("%s creates an IPv4 default route", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_route"
  object.get(after(resource), "cidr_block", "") != ""
}

deny contains sprintf("%s creates an IPv4 default route", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_route_table"
  some route in object.get(after(resource), "route", [])
  object.get(route, "cidr_block", "") != ""
}

deny contains sprintf("%s permits inbound security-group traffic", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type in {"aws_security_group_rule", "aws_vpc_security_group_ingress_rule", "aws_default_security_group"}
}

deny contains sprintf("%s permits inbound security-group traffic", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_security_group"
  count(object.get(after(resource), "ingress", [])) > 0
}

deny contains sprintf("%s permits broad or IPv4 egress", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type in {"aws_security_group_rule", "aws_vpc_security_group_egress_rule", "aws_vpc_security_group_egress_rule"}
  object.get(after(resource), "cidr_ipv4", "") != ""
}

deny contains sprintf("%s permits broad or IPv4 egress", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_vpc_security_group_egress_rule"
  object.get(after(resource), "cidr_ipv6", "") == "::/0"
}

deny contains sprintf("%s permits broad or IPv4 egress", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_security_group"
  some rule in object.get(after(resource), "egress", [])
  count(object.get(rule, "cidr_blocks", [])) > 0
}

deny contains sprintf("%s permits broad or IPv4 egress", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_security_group"
  some rule in object.get(after(resource), "egress", [])
  "::/0" in object.get(rule, "ipv6_cidr_blocks", [])
}

deny contains sprintf("%s must use encrypted gp3 root storage", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_instance"
  root := object.get(after(resource), "root_block_device", [{}])[0]
  object.get(root, "encrypted", false) != true
}

deny contains sprintf("%s must use encrypted gp3 root storage", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_instance"
  root := object.get(after(resource), "root_block_device", [{}])[0]
  object.get(root, "volume_type", "") != "gp3"
}

deny contains sprintf("%s must use a bounded encrypted gp3 root volume", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_instance"
  root := object.get(after(resource), "root_block_device", [{}])[0]
  volume_size := object.get(root, "volume_size", 0)
  volume_size < 8 or volume_size > 64
}

deny contains sprintf("%s must use immutable scanned KMS-encrypted ECR repositories", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_ecr_repository"
  object.get(after(resource), "image_tag_mutability", "") != "IMMUTABLE"
}

deny contains sprintf("%s must use immutable scanned KMS-encrypted ECR repositories", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_ecr_repository"
  object.get(object.get(after(resource), "image_scanning_configuration", [{}])[0], "scan_on_push", false) != true
}

deny contains sprintf("%s must use immutable scanned KMS-encrypted ECR repositories", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_ecr_repository"
  object.get(object.get(after(resource), "encryption_configuration", [{}])[0], "encryption_type", "") != "KMS"
}

deny contains sprintf("%s contains wildcard IAM authority", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type in {"aws_iam_policy", "aws_iam_role_policy"}
  policy := json.unmarshal(object.get(after(resource), "policy", "{}"))
  some statement in policy_statements(policy)
  is_wildcard(object.get(statement, "Action", []))
}

deny contains sprintf("%s contains wildcard IAM authority", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type in {"aws_iam_policy", "aws_iam_role_policy"}
  policy := json.unmarshal(object.get(after(resource), "policy", "{}"))
  some statement in policy_statements(policy)
  is_wildcard(object.get(statement, "Resource", []))
  not wildcard_resource_is_allowed(statement)
}

deny contains sprintf("%s embeds a secret literal in user data or a parameter value", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  raw := sprintf("%v", [after(resource)])
  regex.match("(?i)(-----BEGIN|sk-live-|password=[^$]|token=[A-Za-z0-9_-]{20,})", raw)
}

deny contains sprintf("%s must use a SecureString for connector material", [resource.address]) if {
  some resource in object.get(input, "resource_changes", [])
  managed(resource)
  resource.type == "aws_ssm_parameter"
  object.get(after(resource), "type", "") != "SecureString"
}

deny contains "more than one EC2 instance is planned for the IPv6 trial" if {
  instances := [resource |
    some resource in object.get(input, "resource_changes", [])
    managed(resource)
    resource.type == "aws_instance"
  ]
  count(instances) > 1
}
