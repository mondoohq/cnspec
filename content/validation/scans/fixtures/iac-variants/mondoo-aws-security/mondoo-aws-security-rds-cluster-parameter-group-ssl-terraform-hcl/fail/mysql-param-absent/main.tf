# Non-compliant: MySQL parameter group sets unrelated params but never requires secure transport.
resource "aws_rds_cluster_parameter_group" "fail_example" {
  name   = "example-mysql"
  family = "aurora-mysql8.0"

  parameter {
    name  = "max_connections"
    value = "100"
  }
}
