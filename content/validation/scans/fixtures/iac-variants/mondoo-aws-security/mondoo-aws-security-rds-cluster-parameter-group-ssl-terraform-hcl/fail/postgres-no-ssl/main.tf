# Non-compliant: PostgreSQL parameter group does not enforce SSL.
resource "aws_rds_cluster_parameter_group" "fail_example" {
  name   = "example-postgres"
  family = "aurora-postgresql15"

  parameter {
    name  = "ssl"
    value = "0"
  }
}
