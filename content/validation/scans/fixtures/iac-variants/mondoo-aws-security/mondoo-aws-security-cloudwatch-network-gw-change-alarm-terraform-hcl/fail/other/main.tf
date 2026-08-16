# Non-compliant: metric filter does not match gateway changes.
resource "aws_cloudwatch_log_metric_filter" "fail_example" {
  name           = "unrelated-filter"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = CreateBucket) }"

  metric_transformation {
    name      = "UnrelatedEventCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}

# CloudTrail delivering to CloudWatch Logs; the CIS monitoring checks only apply
# when a trail feeds a metric filter, so the terraform-hcl filter requires both.
resource "aws_cloudtrail" "example" {
  name                       = "example-trail"
  s3_bucket_name             = "example-cloudtrail-logs"
  cloud_watch_logs_group_arn = "arn:aws:logs:us-east-1:123456789012:log-group:example:*"
}
