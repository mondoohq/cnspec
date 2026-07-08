# Non-compliant: log group has no retention set (defaults to never expire).
resource "aws_cloudwatch_log_group" "fail_example" {
  name = "example-log-group"
}
