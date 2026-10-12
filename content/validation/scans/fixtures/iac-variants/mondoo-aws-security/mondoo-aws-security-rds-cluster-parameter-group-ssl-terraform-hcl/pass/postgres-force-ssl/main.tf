# Compliant: PostgreSQL parameter group forces TLS.
resource "aws_rds_cluster_parameter_group" "pass_example" {
  name   = "example-postgres"
  family = "aurora-postgresql15"

  parameter {
    name  = "rds.force_ssl"
    value = "1"
  }
}
