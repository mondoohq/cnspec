# Compliant: parameter group enforces SSL via require_ssl = true.
resource "aws_redshift_parameter_group" "pass_example" {
  name   = "require-ssl-pg"
  family = "redshift-1.0"

  parameter {
    name  = "require_ssl"
    value = "true"
  }
}
