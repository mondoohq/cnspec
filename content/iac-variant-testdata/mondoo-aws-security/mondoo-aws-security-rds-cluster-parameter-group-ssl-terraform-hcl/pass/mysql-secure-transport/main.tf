# Compliant: MySQL parameter group requires secure transport.
resource "aws_rds_cluster_parameter_group" "pass_example" {
  name   = "example-mysql"
  family = "aurora-mysql8.0"

  parameter {
    name  = "require_secure_transport"
    value = "ON"
  }
}
