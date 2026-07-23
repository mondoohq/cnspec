# Compliant: DataSync task logs to a CloudWatch log group.
resource "aws_datasync_task" "pass_example" {
  name                     = "pass-example"
  source_location_arn      = "arn:aws:datasync:us-east-1:123456789012:location/loc-src"
  destination_location_arn = "arn:aws:datasync:us-east-1:123456789012:location/loc-dst"
  cloudwatch_log_group_arn = "arn:aws:logs:us-east-1:123456789012:log-group:/datasync:*"
}
