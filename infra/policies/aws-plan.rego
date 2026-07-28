package aviasurveil360.aws_plan

import rego.v1

required_tags := {
	"CostCenter",
	"DataClassification",
	"Environment",
	"ManagedBy",
	"Owner",
}

tagged_types := {
	"aws_db_instance",
	"aws_iam_policy",
	"aws_launch_template",
	"aws_lb_listener",
	"aws_s3_bucket",
	"aws_vpc_security_group_ingress_rule",
}

data_types := {
	"aws_db_instance",
	"aws_rds_cluster",
	"aws_s3_bucket",
}

changes contains resource if {
	some resource in object.get(input, "resource_changes", [])
	object.get(resource, "mode", "managed") == "managed"
}

after(resource) := object.get(object.get(resource, "change", {}), "after", {})

actions(resource) := object.get(object.get(resource, "change", {}), "actions", [])

tags(resource) := object.get(after(resource), "tags", {})

deny contains sprintf("%s deletes a protected data resource", [resource.address]) if {
	some resource in changes
	resource.type in data_types
	"delete" in actions(resource)
}

deny contains sprintf("%s is missing required tag %s", [resource.address, tag]) if {
	some resource in changes
	resource.type in tagged_types
	some tag in required_tags
	object.get(tags(resource), tag, "") == ""
}

deny contains sprintf("%s exposes a database publicly", [resource.address]) if {
	some resource in changes
	resource.type in {"aws_db_instance", "aws_rds_cluster"}
	object.get(after(resource), "publicly_accessible", false) == true
}

deny contains sprintf("%s leaves database storage unencrypted", [resource.address]) if {
	some resource in changes
	resource.type in {"aws_db_instance", "aws_rds_cluster"}
	object.get(after(resource), "storage_encrypted", false) != true
}

deny contains sprintf("%s disables database deletion protection", [resource.address]) if {
	some resource in changes
	resource.type in {"aws_db_instance", "aws_rds_cluster"}
	object.get(after(resource), "deletion_protection", false) != true
}

deny contains sprintf("%s weakens S3 public access block", [resource.address]) if {
	some resource in changes
	resource.type == "aws_s3_bucket_public_access_block"
	some setting in {
		"block_public_acls",
		"block_public_policy",
		"ignore_public_acls",
		"restrict_public_buckets",
	}
	object.get(after(resource), setting, false) != true
}

deny contains sprintf("%s does not use customer-managed KMS encryption", [resource.address]) if {
	some resource in changes
	resource.type == "aws_s3_bucket_server_side_encryption_configuration"
	some rule in object.get(after(resource), "rule", [])
	some encryption_default in object.get(rule, "apply_server_side_encryption_by_default", [])
	object.get(encryption_default, "sse_algorithm", "") != "aws:kms"
}

deny contains sprintf("%s omits a KMS key", [resource.address]) if {
	some resource in changes
	resource.type == "aws_s3_bucket_server_side_encryption_configuration"
	some rule in object.get(after(resource), "rule", [])
	some encryption_default in object.get(rule, "apply_server_side_encryption_by_default", [])
	not object.get(encryption_default, "kms_master_key_id", "")
}

deny contains sprintf("%s is not an HTTPS-only listener", [resource.address]) if {
	some resource in changes
	resource.type == "aws_lb_listener"
	object.get(after(resource), "protocol", "") != "HTTPS"
}

deny contains sprintf("%s is not on TCP 443", [resource.address]) if {
	some resource in changes
	resource.type == "aws_lb_listener"
	object.get(after(resource), "port", 0) != 443
}

public_cidr(resource) if {
	object.get(after(resource), "cidr_ipv4", "") == "0.0.0.0/0"
}

public_cidr(resource) if {
	object.get(after(resource), "cidr_ipv6", "") == "::/0"
}

approved_alb_https(resource) if {
	endswith(resource.address, ".alb_https")
	object.get(after(resource), "ip_protocol", "") == "tcp"
	object.get(after(resource), "from_port", 0) == 443
	object.get(after(resource), "to_port", 0) == 443
}

deny contains sprintf("%s exposes non-ALB ingress publicly", [resource.address]) if {
	some resource in changes
	resource.type in {
		"aws_security_group_rule",
		"aws_vpc_security_group_ingress_rule",
	}
	public_cidr(resource)
	not approved_alb_https(resource)
}

statements(policy) := object.get(policy, "Statement", []) if {
	is_array(object.get(policy, "Statement", []))
}

statements(policy) := [object.get(policy, "Statement", {})] if {
	is_object(object.get(policy, "Statement", {}))
}

wildcard_action(statement) if {
	object.get(statement, "Action", "") == "*"
}

wildcard_action(statement) if {
	some action in object.get(statement, "Action", [])
	contains(action, "*")
}

deny contains sprintf("%s grants a wildcard IAM action", [resource.address]) if {
	some resource in changes
	resource.type == "aws_iam_policy"
	policy := json.unmarshal(object.get(after(resource), "policy", "{}"))
	some statement in statements(policy)
	wildcard_action(statement)
}

deny contains sprintf("%s lacks an immutable image digest", [resource.address]) if {
	some resource in changes
	resource.type == "aws_launch_template"
	digest := object.get(tags(resource), "ImageDigest", "")
	not regex.match("^[a-f0-9]{64}$", digest)
}

deny contains sprintf("%s has an unencrypted root volume", [resource.address]) if {
	some resource in changes
	resource.type == "aws_launch_template"
	some mapping in object.get(after(resource), "block_device_mappings", [])
	some ebs in object.get(mapping, "ebs", [])
	encrypted := object.get(ebs, "encrypted", false)
	encrypted != true
	encrypted != "true"
}

deny contains sprintf("%s permits optional instance metadata tokens", [resource.address]) if {
	some resource in changes
	resource.type == "aws_launch_template"
	some options in object.get(after(resource), "metadata_options", [])
	object.get(options, "http_tokens", "") != "required"
}
