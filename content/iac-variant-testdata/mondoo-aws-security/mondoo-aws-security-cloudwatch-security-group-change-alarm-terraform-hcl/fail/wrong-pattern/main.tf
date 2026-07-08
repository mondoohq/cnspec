# Non-compliant: metric filter does not match the required API event.
resource "aws_cloudwatch_log_metric_filter" "fail_example" {
  name           = "example-filter"
  log_group_name = "example-log-group"
  pattern        = "{ ($.eventName = CreateBucket*) }"

  metric_transformation {
    name      = "EventCount"
    namespace = "CloudTrailMetrics"
    value     = "1"
  }
}
