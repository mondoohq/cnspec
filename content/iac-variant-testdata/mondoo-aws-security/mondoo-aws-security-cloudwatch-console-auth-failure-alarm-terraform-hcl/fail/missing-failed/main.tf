# Non-compliant: metric filter matches console logins but not failed attempts.
resource "aws_cloudwatch_log_metric_filter" "fail_example" {
  name           = "console-logins"
  log_group_name = "example"
  pattern        = "{ ($.eventName = ConsoleLogin) }"

  metric_transformation {
    name      = "ConsoleLogins"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
