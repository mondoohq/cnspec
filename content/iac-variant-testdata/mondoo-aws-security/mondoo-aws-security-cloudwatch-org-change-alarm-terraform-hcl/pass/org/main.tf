# Compliant: metric filter matches AWS Organizations change API calls.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "org-changes"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventSource = organizations.amazonaws.com) && (($.eventName = AcceptHandshake) || ($.eventName = CreateAccount)) }"

  metric_transformation {
    name      = "OrganizationEventCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
