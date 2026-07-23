# Non-compliant: DataSync task has no CloudWatch log group configured.
resource "aws_datasync_task" "fail_example" {
  name                     = "fail-example"
  source_location_arn      = "arn:aws:datasync:us-east-1:123456789012:location/loc-src"
  destination_location_arn = "arn:aws:datasync:us-east-1:123456789012:location/loc-dst"
}
