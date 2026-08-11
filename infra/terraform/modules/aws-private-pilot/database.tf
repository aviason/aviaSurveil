resource "aws_db_subnet_group" "this" {
  name_prefix = "${var.name}-"
  description = "Two-AZ structural subnet group for one Single-AZ private-pilot RDS instance"
  subnet_ids  = [for key in ["a", "b"] : aws_subnet.private_database[key].id]

  tags = merge(var.tags, { Availability = "single-az-database-two-az-subnet-group" })
}

resource "aws_db_parameter_group" "this" {
  name_prefix = "${var.name}-"
  family      = "postgres${split(".", var.database_engine_version)[0]}"
  description = "Bounded private-pilot PostgreSQL connections and mandatory TLS"

  parameter {
    name  = "max_connections"
    value = "50"
  }

  parameter {
    name  = "rds.force_ssl"
    value = "1"
  }

  tags = var.tags
}

resource "aws_db_instance" "this" {
  identifier_prefix = "${var.name}-"

  engine         = "postgres"
  engine_version = var.database_engine_version
  instance_class = "db.t4g.micro"
  port           = 5432

  username                      = "pilotbootstrap"
  manage_master_user_password   = true
  master_user_secret_kms_key_id = aws_kms_key.secrets.arn

  allocated_storage     = var.database_allocated_storage_gib
  max_allocated_storage = 100
  storage_type          = "gp3"
  storage_encrypted     = true
  kms_key_id            = aws_kms_key.data.arn

  availability_zone      = var.availability_zones[0]
  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.database.id]
  publicly_accessible    = false
  multi_az               = false

  iam_database_authentication_enabled = true

  parameter_group_name = aws_db_parameter_group.this.name

  backup_retention_period = 14
  backup_window           = "01:00-02:00"
  maintenance_window      = "sun:03:00-sun:04:00"

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
  performance_insights_enabled    = true
  performance_insights_kms_key_id = aws_kms_key.data.arn
  auto_minor_version_upgrade      = true
  apply_immediately               = false

  deletion_protection       = true
  skip_final_snapshot       = false
  final_snapshot_identifier = "${var.name}-final"
  copy_tags_to_snapshot     = true

  tags = merge(var.tags, {
    Name             = "${var.name}-postgresql"
    Availability     = "single-az-non-ha"
    LogicalDatabases = "aviasurveil360,keycloak"
  })

  lifecycle {
    prevent_destroy = true
    precondition {
      condition     = var.database_allocated_storage_gib <= 90
      error_message = "Private-pilot storage autoscaling must remain bounded."
    }
  }
}
