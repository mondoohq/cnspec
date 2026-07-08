# Compliant: metric filter watches for AWS Config changes.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "config-changes"
  log_group_name = "example"
  pattern        = "{ ($.eventName = StopConfigurationRecorder) || ($.eventName = DeleteDeliveryChannel) }"

  metric_transformation {
    name      = "ConfigChanges"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
