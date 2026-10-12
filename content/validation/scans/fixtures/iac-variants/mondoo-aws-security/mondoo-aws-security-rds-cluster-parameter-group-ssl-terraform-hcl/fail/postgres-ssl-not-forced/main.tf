# Non-compliant: ssl only makes TLS available; without rds.force_ssl, plaintext connections are accepted.
resource "aws_rds_cluster_parameter_group" "fail_example" {
  name   = "example-postgres"
  family = "aurora-postgresql15"

  parameter {
    name  = "ssl"
    value = "1"
  }
}
