# Non-compliant: log group has no KMS key configured.
resource "aws_cloudwatch_log_group" "fail_example" {
  name = "example-log-group"
}
