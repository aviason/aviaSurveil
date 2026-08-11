locals {
  environment         = "production"
  owner               = get_env("AVIA_AWS_PRIVATE_PILOT_OWNER", "")
  cost_center         = get_env("AVIA_AWS_PRIVATE_PILOT_COST_CENTER", "")
  data_classification = get_env("AVIA_AWS_PRIVATE_PILOT_DATA_CLASSIFICATION", "")
}
