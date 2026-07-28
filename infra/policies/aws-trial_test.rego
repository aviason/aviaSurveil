package aviasurveil360.aws_trial

import rego.v1

fixture_change(type, after) := {
	"address": sprintf("%s.fixture", [type]),
	"mode": "managed",
	"type": type,
	"change": {
		"actions": ["create"],
		"after": after,
		"after_unknown": {},
	},
}

tags := {
	"Environment": "fixture",
	"Owner": "platform-operations",
	"CostCenter": "trial-001",
	"DataClassification": "restricted",
	"ManagedBy": "terraform",
}

test_secure_fixture_has_no_denials if {
	fixture_input := {"resource_changes": [
		fixture_change("aws_db_instance", {
			"publicly_accessible": false,
			"storage_encrypted": true,
			"deletion_protection": true,
			"skip_final_snapshot": false,
			"kms_key_id": "arn:aws:kms:fixture",
			"tags": tags,
		}),
		fixture_change("aws_ecr_repository", {
			"image_tag_mutability": "IMMUTABLE",
			"image_scanning_configuration": [{"scan_on_push": true}],
			"encryption_configuration": [{"encryption_type": "KMS"}],
			"tags": tags,
		}),
		fixture_change("aws_s3_bucket_public_access_block", {
			"block_public_acls": true,
			"block_public_policy": true,
			"ignore_public_acls": true,
			"restrict_public_buckets": true,
		}),
		fixture_change("aws_s3_bucket_versioning", {"versioning_configuration": [{"status": "Enabled"}]}),
	]}
	messages := deny with input as fixture_input
	count(messages) == 0
}

test_public_database_is_denied if {
	fixture_input := {"resource_changes": [fixture_change("aws_db_instance", {
		"publicly_accessible": true,
		"storage_encrypted": true,
		"deletion_protection": true,
		"skip_final_snapshot": false,
		"kms_key_id": "arn:aws:kms:fixture",
		"tags": tags,
	})]}
	messages := deny with input as fixture_input
	some message in messages
	contains(message, "public database")
}

test_missing_tag_is_denied if {
	fixture_input := {"resource_changes": [fixture_change("aws_vpc", {"tags": object.remove(tags, {"CostCenter"})})]}
	messages := deny with input as fixture_input
	some message in messages
	contains(message, "CostCenter")
}

test_wildcard_iam_is_denied if {
	fixture_input := {"resource_changes": [fixture_change("aws_iam_policy", {
		"policy": json.marshal({
			"Version": "2012-10-17",
			"Statement": [{"Effect": "Allow", "Action": "*", "Resource": "*"}],
		}),
		"tags": tags,
	})]}
	messages := deny with input as fixture_input
	some message in messages
	contains(message, "wildcard IAM")
}

test_public_compute_is_denied if {
	fixture_input := {"resource_changes": [fixture_change("aws_launch_template", {
		"metadata_options": [{"http_tokens": "required"}],
		"network_interfaces": [{"associate_public_ip_address": true}],
		"block_device_mappings": [{"ebs": [{"encrypted": true}]}],
		"user_data": base64.encode("#!/bin/sh\ndocker pull repo/image@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		"tags": tags,
	})]}
	messages := deny with input as fixture_input
	some message in messages
	contains(message, "public compute address")
}

test_plaintext_secret_resource_is_denied if {
	fixture_input := {"resource_changes": [fixture_change("aws_secretsmanager_secret_version", {"secret_string": "fixture-plaintext"})]}
	messages := deny with input as fixture_input
	some message in messages
	contains(message, "secret value")
}
