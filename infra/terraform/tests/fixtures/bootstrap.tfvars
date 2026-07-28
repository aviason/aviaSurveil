region            = "eu-central-1"
state_bucket_name = "avia-fixture-terraform-state-111122223333"
kms_alias         = "alias/avia-fixture-terraform-state"

tags = {
  Environment        = "fixture"
  Owner              = "platform-operations"
  CostCenter         = "trial-001"
  DataClassification = "restricted"
  ManagedBy          = "terraform"
}
