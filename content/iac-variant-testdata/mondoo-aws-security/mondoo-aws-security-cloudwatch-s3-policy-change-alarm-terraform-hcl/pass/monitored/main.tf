# Compliant: metric filter matches the required API event.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "example-filter"
  log_group_name = "example-log-group"
  pattern        = "{ ($.eventName = PutBucketPolicy*) }"

  metric_transformation {
    name      = "EventCount"
    namespace = "CloudTrailMetrics"
    value     = "1"
  }
}
