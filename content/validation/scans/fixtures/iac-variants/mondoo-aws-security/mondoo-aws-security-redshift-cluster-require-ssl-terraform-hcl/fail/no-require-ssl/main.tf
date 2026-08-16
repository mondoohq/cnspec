# Non-compliant: parameter group sets other parameters but never require_ssl,
# so SSL is not enforced.
resource "aws_redshift_parameter_group" "fail_example" {
  name   = "no-require-ssl-pg"
  family = "redshift-1.0"

  parameter {
    name  = "enable_user_activity_logging"
    value = "true"
  }
}
