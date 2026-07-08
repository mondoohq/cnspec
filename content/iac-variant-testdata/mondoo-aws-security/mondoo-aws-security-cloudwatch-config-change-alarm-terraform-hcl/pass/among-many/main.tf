# Compliant: among several metric filters, one watches for AWS Config changes.
resource "aws_cloudwatch_log_metric_filter" "console_signin_failures" {
  name           = "console-signin-failures"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = ConsoleLogin) && ($.errorMessage = \"Failed authentication\") }"

  metric_transformation {
    name      = "ConsoleSigninFailures"
    namespace = "CISBenchmark"
    value     = "1"
  }
}

resource "aws_cloudwatch_log_metric_filter" "config_changes" {
  name           = "aws-config-changes"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventSource = config.amazonaws.com) && (($.eventName = StopConfigurationRecorder) || ($.eventName = DeleteDeliveryChannel) || ($.eventName = PutDeliveryChannel) || ($.eventName = PutConfigurationRecorder)) }"

  metric_transformation {
    name      = "AWSConfigChanges"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
