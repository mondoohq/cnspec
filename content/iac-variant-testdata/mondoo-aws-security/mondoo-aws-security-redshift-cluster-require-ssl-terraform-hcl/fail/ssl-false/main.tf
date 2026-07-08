# Non-compliant: require_ssl is set to false, so SSL is not enforced.
resource "aws_redshift_parameter_group" "fail_example" {
  name   = "no-ssl-pg"
  family = "redshift-1.0"

  parameter {
    name  = "require_ssl"
    value = "false"
  }
}
