mock_provider "aws" {}

run "remote_state_is_encrypted_versioned_locked_and_protected" {
  command = plan

  module {
    source = "./bootstrap/remote-state"
  }

  variables {
    state_bucket_name = "avia-fixture-terraform-state-111122223333"
    kms_alias         = "alias/avia-fixture-terraform-state"
    tags = {
      Environment        = "fixture"
      Owner              = "platform-operations"
      CostCenter         = "trial-001"
      DataClassification = "restricted"
      ManagedBy          = "terraform"
    }
  }

  assert {
    condition     = aws_kms_key.state.enable_key_rotation
    error_message = "Remote-state KMS encryption must rotate."
  }

  assert {
    condition     = aws_s3_bucket_versioning.state.versioning_configuration[0].status == "Enabled"
    error_message = "Remote state must retain versions."
  }

  assert {
    condition = alltrue([
      aws_s3_bucket_public_access_block.state.block_public_acls,
      aws_s3_bucket_public_access_block.state.block_public_policy,
      aws_s3_bucket_public_access_block.state.ignore_public_acls,
      aws_s3_bucket_public_access_block.state.restrict_public_buckets,
    ])
    error_message = "Remote state must block all public access."
  }

  assert {
    condition = one(
      one(aws_s3_bucket_server_side_encryption_configuration.state.rule).apply_server_side_encryption_by_default
    ).sse_algorithm == "aws:kms"
    error_message = "Remote state must use KMS encryption."
  }
}
