# Compliant: multi-region trail forwards to a CloudWatch Logs group.
resource "aws_cloudtrail" "pass_example" {
  name                          = "example"
  s3_bucket_name                = "example-bucket"
  is_multi_region_trail         = true
  cloud_watch_logs_group_arn    = "arn:aws:logs:us-east-1:123456789012:log-group:example:*"
  cloud_watch_logs_role_arn     = "arn:aws:iam::123456789012:role/example"
}
