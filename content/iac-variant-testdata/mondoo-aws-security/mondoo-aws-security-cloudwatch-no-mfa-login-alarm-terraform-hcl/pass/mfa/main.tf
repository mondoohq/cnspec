# Compliant: metric filter matches console logins without MFA.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "no-mfa-console-login"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = ConsoleLogin) && ($.additionalEventData.MFAUsed != Yes) }"

  metric_transformation {
    name      = "NoMFAConsoleLoginCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
