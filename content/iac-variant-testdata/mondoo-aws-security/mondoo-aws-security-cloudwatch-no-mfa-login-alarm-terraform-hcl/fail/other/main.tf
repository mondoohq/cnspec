# Non-compliant: metric filter does not check for MFA usage.
resource "aws_cloudwatch_log_metric_filter" "fail_example" {
  name           = "console-login"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = ConsoleLogin) }"

  metric_transformation {
    name      = "ConsoleLoginCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
