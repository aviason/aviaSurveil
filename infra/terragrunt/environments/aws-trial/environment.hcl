locals {
  environment         = "aws-trial"
  owner               = get_env("AVIA_AWS_OWNER", "")
  cost_center         = get_env("AVIA_AWS_COST_CENTER", "")
  data_classification = get_env("AVIA_AWS_DATA_CLASSIFICATION", "")
  change_window       = get_env("AVIA_AWS_CHANGE_WINDOW", "")
}
