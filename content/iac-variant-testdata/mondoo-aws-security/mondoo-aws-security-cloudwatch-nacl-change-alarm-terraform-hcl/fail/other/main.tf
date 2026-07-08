# Non-compliant: metric filter does not match network ACL changes.
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
