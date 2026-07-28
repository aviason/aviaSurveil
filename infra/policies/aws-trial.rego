package aviasurveil360.aws_trial

import rego.v1

required_tags := {
	"Environment",
	"Owner",
	"CostCenter",
	"DataClassification",
	"ManagedBy",
}

tagged_types := {
	"aws_autoscaling_group",
	"aws_backup_plan",
	"aws_backup_vault",
	"aws_cloudwatch_log_group",
	"aws_cloudwatch_metric_alarm",
	"aws_db_instance",
	"aws_db_subnet_group",
	"aws_ecr_repository",
	"aws_eip",
	"aws_iam_instance_profile",
	"aws_iam_policy",
	"aws_iam_role",
	"aws_internet_gateway",
	"aws_kms_key",
	"aws_launch_template",
	"aws_lb",
	"aws_lb_listener",
	"aws_lb_target_group",
	"aws_nat_gateway",
	"aws_route_table",
	"aws_s3_bucket",
	"aws_secretsmanager_secret",
	"aws_security_group",
	"aws_ssm_parameter",
	"aws_subnet",
	"aws_vpc",
	"aws_vpc_endpoint",
}

managed(r) if {
	r.mode == "managed"
	r.change.actions != ["delete"]
}

after(r) := object.get(r.change, "after", {})

after_unknown(r) := object.get(r.change, "after_unknown", {})

resource_tags(r) := object.get(after(r), "tags", {}) if {
	r.type != "aws_autoscaling_group"
}

resource_tags(r) := {item.key: item.value | some item in object.get(after(r), "tag", [])} if {
	r.type == "aws_autoscaling_group"
}

attribute_present(r, name) if {
	object.get(after(r), name, null) != null
}

attribute_present(r, name) if {
	object.get(after_unknown(r), name, false) == true
}

is_wildcard(value) if {
	value == "*"
}

is_true(value) if {
	value == true
}

is_true(value) if {
	value == "true"
}

is_false(value) if {
	value == false
}

is_false(value) if {
	value == "false"
}

is_wildcard(value) if {
	is_array(value)
	some item in value
	item == "*"
}

policy_statements(policy) := policy.Statement if {
	is_array(policy.Statement)
}

policy_statements(policy) := [policy.Statement] if {
	is_object(policy.Statement)
}

deny contains sprintf("%s is missing mandatory tag %s", [r.address, tag]) if {
	some r in input.resource_changes
	managed(r)
	r.type in tagged_types
	some tag in required_tags
	tags := resource_tags(r)
	trim_space(object.get(tags, tag, "")) == ""
}

deny contains sprintf("%s exposes a public database", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_db_instance"
	object.get(after(r), "publicly_accessible", true) != false
}

deny contains sprintf("%s must encrypt database storage", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_db_instance"
	object.get(after(r), "storage_encrypted", false) != true
}

deny contains sprintf("%s must enable deletion protection", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_db_instance"
	object.get(after(r), "deletion_protection", false) != true
}

deny contains sprintf("%s must retain a final snapshot", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_db_instance"
	object.get(after(r), "skip_final_snapshot", true) != false
}

deny contains sprintf("%s must use an explicit KMS key", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type in {
		"aws_backup_vault",
		"aws_cloudwatch_log_group",
		"aws_db_instance",
		"aws_secretsmanager_secret",
	}
	not attribute_present(r, "kms_key_arn")
	not attribute_present(r, "kms_key_id")
}

deny contains sprintf("%s must rotate its KMS key", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_kms_key"
	object.get(after(r), "enable_key_rotation", false) != true
}

deny contains sprintf("%s must require IMDSv2", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_launch_template"
	metadata := object.get(after(r), "metadata_options", [{}])[0]
	object.get(metadata, "http_tokens", "") != "required"
}

deny contains sprintf("%s assigns a public compute address", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_launch_template"
	interface := object.get(after(r), "network_interfaces", [{}])[0]
	not is_false(object.get(interface, "associate_public_ip_address", true))
}

deny contains sprintf("%s must encrypt its root volume", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_launch_template"
	mapping := object.get(after(r), "block_device_mappings", [{}])[0]
	ebs := object.get(mapping, "ebs", [{}])[0]
	not is_true(object.get(ebs, "encrypted", false))
}

deny contains sprintf("%s must use an immutable ECR digest", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_launch_template"
	tags := resource_tags(r)
	not regex.match("^[0-9a-f]{64}$", object.get(tags, "ImageDigest", ""))
}

deny contains sprintf("%s permits public S3 access", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_s3_bucket_public_access_block"
	some key in {"block_public_acls", "block_public_policy", "ignore_public_acls", "restrict_public_buckets"}
	object.get(after(r), key, false) != true
}

deny contains sprintf("%s must enable S3 versioning", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_s3_bucket_versioning"
	config := object.get(after(r), "versioning_configuration", [{}])[0]
	object.get(config, "status", "") != "Enabled"
}

deny contains sprintf("%s must use immutable ECR tags", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_ecr_repository"
	object.get(after(r), "image_tag_mutability", "") != "IMMUTABLE"
}

deny contains sprintf("%s must scan ECR images on push", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_ecr_repository"
	config := object.get(after(r), "image_scanning_configuration", [{}])[0]
	object.get(config, "scan_on_push", false) != true
}

deny contains sprintf("%s must use KMS ECR encryption", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_ecr_repository"
	config := object.get(after(r), "encryption_configuration", [{}])[0]
	object.get(config, "encryption_type", "") != "KMS"
}

deny contains sprintf("%s exposes a non-HTTPS load balancer listener", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_lb_listener"
	object.get(after(r), "port", 0) != 443
}

deny contains sprintf("%s exposes a non-HTTPS load balancer listener", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_lb_listener"
	object.get(after(r), "protocol", "") != "HTTPS"
}

deny contains sprintf("%s exposes SSH ingress", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_vpc_security_group_ingress_rule"
	object.get(after(r), "from_port", 0) <= 22
	object.get(after(r), "to_port", 0) >= 22
}

deny contains sprintf("%s contains wildcard IAM authority", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_iam_policy"
	raw := object.get(after(r), "policy", "{}")
	policy := json.unmarshal(raw)
	some statement in policy_statements(policy)
	is_wildcard(object.get(statement, "Action", []))
}

deny contains sprintf("%s contains wildcard IAM authority", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_iam_policy"
	raw := object.get(after(r), "policy", "{}")
	policy := json.unmarshal(raw)
	some statement in policy_statements(policy)
	is_wildcard(object.get(statement, "Resource", []))
}

deny contains sprintf("%s embeds a secret value in Terraform", [r.address]) if {
	some r in input.resource_changes
	managed(r)
	r.type == "aws_secretsmanager_secret_version"
}
