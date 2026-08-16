# Compliant: log group has a positive retention period.
resource "aws_cloudwatch_log_group" "pass_example" {
  name              = "example-log-group"
  retention_in_days = 90
}
