package aviasurveil360.aws_private_pilot

import rego.v1

fixture_change(resource_type, name, value) := {
  "address": sprintf("%s.%s", [resource_type, name]),
  "mode": "managed",
  "type": resource_type,
  "change": {
    "actions": ["create"],
    "after": value,
  },
}

secure_fixture := {"resource_changes": [
  fixture_change("aws_instance", "runtime", {
    "instance_type": "t4g.small",
    "associate_public_ip_address": false,
    "ipv6_address_count": 1,
    "metadata_options": [{"http_tokens": "required"}],
    "root_block_device": [{"encrypted": true, "volume_type": "gp3"}],
  }),
  fixture_change("aws_egress_only_internet_gateway", "this", {}),
  fixture_change("aws_db_instance", "this", {
    "instance_class": "db.t4g.micro",
    "multi_az": false,
    "publicly_accessible": false,
    "storage_encrypted": true,
    "backup_retention_period": 14,
    "deletion_protection": true,
    "skip_final_snapshot": false,
  }),
  fixture_change("aws_vpc_endpoint", "s3", {"vpc_endpoint_type": "Gateway"}),
  fixture_change("aws_s3_bucket_public_access_block", "objects", {
    "block_public_acls": true,
    "block_public_policy": true,
    "ignore_public_acls": true,
    "restrict_public_buckets": true,
  }),
  fixture_change("aws_s3_bucket_versioning", "objects", {"versioning_configuration": [{"status": "Enabled"}]}),
  fixture_change("aws_guardduty_malware_protection_plan", "quarantine", {"actions": [{"tagging": [{"status": "ENABLED"}]}]}),
  fixture_change("aws_ecr_repository", "runtime", {
    "image_tag_mutability": "IMMUTABLE",
    "image_scanning_configuration": [{"scan_on_push": true}],
  }),
  fixture_change("cloudflare_zero_trust_tunnel_cloudflared", "this", {}),
  fixture_change("cloudflare_zero_trust_tunnel_cloudflared_config", "this", {}),
  fixture_change("cloudflare_dns_record", "application", {"proxied": true, "type": "CNAME"}),
  fixture_change("aws_ssm_parameter", "cloudflare_connector", {"type": "SecureString", "tier": "Standard"}),
  fixture_change("aws_budgets_budget", "this", {}),
]}

test_secure_fixture_has_no_denials if {
  messages := deny with input as secure_fixture
  count(messages) == 0
}

pre_cutover_fixture := {"resource_changes": [change |
  some change in secure_fixture.resource_changes
  change.type != "cloudflare_dns_record"
]}

test_pre_cutover_fixture_allows_no_dns_record if {
  messages := deny with input as pre_cutover_fixture
  count(messages) == 0
}

test_unproxied_dns_cutover_is_denied if {
  unproxied := fixture_change("cloudflare_dns_record", "application", {"proxied": false, "type": "CNAME"})
  without_dns := pre_cutover_fixture.resource_changes
  mutated := {"resource_changes": array.concat(without_dns, [unproxied])}
  messages := deny with input as mutated
  some message in messages
  contains(message, "must remain Cloudflare-proxied")
}

test_second_compute_is_denied if {
  mutated := object.union(secure_fixture, {"resource_changes": array.concat(secure_fixture.resource_changes, [fixture_change("aws_instance", "second", {
    "instance_type": "t4g.small",
    "associate_public_ip_address": false,
    "ipv6_address_count": 1,
    "metadata_options": [{"http_tokens": "required"}],
    "root_block_device": [{"encrypted": true, "volume_type": "gp3"}],
  })])})
  messages := deny with input as mutated
  "the private pilot must contain exactly one EC2 instance" in messages
}

test_multi_az_rds_is_denied if {
  mutated_database := fixture_change("aws_db_instance", "this", {
    "instance_class": "db.t4g.micro",
    "multi_az": true,
    "publicly_accessible": false,
    "storage_encrypted": true,
    "backup_retention_period": 14,
    "deletion_protection": true,
    "skip_final_snapshot": false,
  })
  mutated := object.union(secure_fixture, {"resource_changes": array.concat(array.slice(secure_fixture.resource_changes, 0, 2), array.concat([mutated_database], array.slice(secure_fixture.resource_changes, 3, count(secure_fixture.resource_changes))))})
  messages := deny with input as mutated
  some message in messages
  contains(message, "Single-AZ db.t4g.micro")
}

test_interface_endpoint_is_denied if {
  mutated_endpoint := fixture_change("aws_vpc_endpoint", "s3", {"vpc_endpoint_type": "Interface"})
  mutated := object.union(secure_fixture, {"resource_changes": array.concat(array.slice(secure_fixture.resource_changes, 0, 3), array.concat([mutated_endpoint], array.slice(secure_fixture.resource_changes, 4, count(secure_fixture.resource_changes))))})
  messages := deny with input as mutated
  some message in messages
  contains(message, "interface-endpoint sprawl")
}

test_secret_value_resource_is_denied if {
  mutated := object.union(secure_fixture, {"resource_changes": array.concat(secure_fixture.resource_changes, [fixture_change("aws_secretsmanager_secret_version", "forbidden", {})])})
  messages := deny with input as mutated
  some message in messages
  contains(message, "forbidden private-pilot resource")
}
