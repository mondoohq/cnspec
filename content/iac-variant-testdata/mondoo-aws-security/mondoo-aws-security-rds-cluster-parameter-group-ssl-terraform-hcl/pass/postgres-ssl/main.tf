# Compliant: PostgreSQL parameter group enforces SSL.
resource "aws_rds_cluster_parameter_group" "pass_example" {
  name   = "example-postgres"
  family = "aurora-postgresql15"

  parameter {
    name  = "ssl"
    value = "1"
  }
}
