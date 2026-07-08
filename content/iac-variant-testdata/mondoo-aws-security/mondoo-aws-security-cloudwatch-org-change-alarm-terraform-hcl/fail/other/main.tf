# Non-compliant: metric filter does not monitor AWS Organizations.
resource "aws_cloudwatch_log_metric_filter" "fail_example" {
  name           = "unrelated-filter"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventSource = s3.amazonaws.com) }"

  metric_transformation {
    name      = "UnrelatedEventCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
