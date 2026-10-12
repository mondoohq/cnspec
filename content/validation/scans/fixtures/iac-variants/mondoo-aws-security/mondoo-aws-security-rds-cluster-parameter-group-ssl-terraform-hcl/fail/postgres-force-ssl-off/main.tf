# Non-compliant: PostgreSQL parameter group does not force TLS.
resource "aws_rds_cluster_parameter_group" "fail_example" {
  name   = "example-postgres"
  family = "aurora-postgresql15"

  parameter {
    name  = "rds.force_ssl"
    value = "0"
  }
}
