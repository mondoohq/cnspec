# Compliant: metric filter watches for failed console authentication attempts.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "console-auth-failures"
  log_group_name = "example"
  pattern        = "{ ($.eventName = ConsoleLogin) && ($.errorMessage = \"Failed authentication\") }"

  metric_transformation {
    name      = "ConsoleAuthFailures"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
