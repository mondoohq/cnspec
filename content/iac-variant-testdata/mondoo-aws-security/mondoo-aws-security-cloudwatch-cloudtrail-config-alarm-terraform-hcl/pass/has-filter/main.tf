# Compliant: metric filter watches for CloudTrail configuration changes.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "cloudtrail-config-changes"
  log_group_name = "example"
  pattern        = "{ ($.eventName = CreateTrail) || ($.eventName = UpdateTrail) || ($.eventName = DeleteTrail) || ($.eventName = StartLogging) || ($.eventName = StopLogging) }"

  metric_transformation {
    name      = "CloudTrailConfigChanges"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
