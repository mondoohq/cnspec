# Non-compliant: metric filter does not monitor root account usage.
resource "aws_cloudwatch_log_metric_filter" "fail_example" {
  name           = "unrelated-filter"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = ConsoleLogin) }"

  metric_transformation {
    name      = "ConsoleLoginCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
