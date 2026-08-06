locals {
  environment         = "aws-ipv6-trial"
  owner               = get_env("AVIA_AWS_IPV6_TRIAL_OWNER", "")
  cost_center         = get_env("AVIA_AWS_IPV6_TRIAL_COST_CENTER", "")
  data_classification = get_env("AVIA_AWS_IPV6_TRIAL_DATA_CLASSIFICATION", "")
  change_window       = get_env("AVIA_AWS_IPV6_TRIAL_CHANGE_WINDOW", "")
}
