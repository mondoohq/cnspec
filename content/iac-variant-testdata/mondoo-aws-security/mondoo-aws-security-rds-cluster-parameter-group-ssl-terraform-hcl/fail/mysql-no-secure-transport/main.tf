# Non-compliant: MySQL parameter group does not require secure transport.
resource "aws_rds_cluster_parameter_group" "fail_example" {
  name   = "example-mysql"
  family = "aurora-mysql8.0"

  parameter {
    name  = "require_secure_transport"
    value = "OFF"
  }
}
