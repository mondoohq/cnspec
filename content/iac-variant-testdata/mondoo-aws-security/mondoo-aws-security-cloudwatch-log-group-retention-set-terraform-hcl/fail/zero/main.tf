# Non-compliant: retention explicitly set to 0, which means logs never expire.
resource "aws_cloudwatch_log_group" "fail_example" {
  name              = "example-log-group"
  retention_in_days = 0
}
