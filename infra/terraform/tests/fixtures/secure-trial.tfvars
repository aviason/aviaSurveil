name_prefix = "avia"
environment = "fixture"
region      = "eu-central-1"

availability_zones    = ["eu-central-1a", "eu-central-1b"]
vpc_cidr              = "10.42.0.0/16"
public_subnet_cidrs   = ["10.42.0.0/24", "10.42.1.0/24"]
compute_subnet_cidrs  = ["10.42.10.0/24", "10.42.11.0/24"]
database_subnet_cidrs = ["10.42.20.0/24", "10.42.21.0/24"]
enable_nat_gateway    = false
single_nat_gateway    = false

certificate_arn    = "arn:aws:acm:eu-central-1:111122223333:certificate/11111111-2222-3333-4444-555555555555"
alarm_topic_arn    = "arn:aws:sns:eu-central-1:111122223333:avia-fixture-alerts"
otel_endpoint      = "http://127.0.0.1:4318"
bucket_name_prefix = "avia-fixture-111122223333"

repositories                = ["runtime"]
secret_names                = ["application-runtime", "identity-runtime"]
secret_recovery_window_days = 30

application_port = 8443
ami_id           = "ami-0123456789abcdef0"
image_uri        = "111122223333.dkr.ecr.eu-central-1.amazonaws.com/avia-fixture-runtime@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
instance_type    = "t4g.medium"
min_size         = 2
desired_capacity = 2
max_size         = 4

database_instance_class    = "db.t4g.medium"
database_engine_version    = "17.6"
database_allocated_storage = 50
backup_retention_days      = 30
log_retention_days         = 30

tags = {
  Environment        = "fixture"
  Owner              = "platform-operations"
  CostCenter         = "trial-001"
  DataClassification = "restricted"
  ManagedBy          = "terraform"
}
