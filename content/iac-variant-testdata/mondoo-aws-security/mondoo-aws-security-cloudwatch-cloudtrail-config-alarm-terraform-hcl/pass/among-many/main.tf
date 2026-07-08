# Compliant: among several metric filters, one watches for CloudTrail
# configuration changes.
resource "aws_cloudwatch_log_metric_filter" "unauthorized_api" {
  name           = "unauthorized-api-calls"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") }"

  metric_transformation {
    name      = "UnauthorizedAPICalls"
    namespace = "CISBenchmark"
    value     = "1"
  }
}

resource "aws_cloudwatch_log_metric_filter" "cloudtrail_config" {
  name           = "cloudtrail-config-changes"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = CreateTrail) || ($.eventName = UpdateTrail) || ($.eventName = DeleteTrail) || ($.eventName = StartLogging) || ($.eventName = StopLogging) }"

  metric_transformation {
    name      = "CloudTrailConfigChanges"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
