# Compliant: CIS unauthorized API pattern with AccessDenied.
resource "aws_cloudwatch_log_metric_filter" "unauthorized_api" {
  name           = "unauthorized_api"
  log_group_name = "example-log-group"
  pattern        = "{ (\$.errorCode = \"*UnauthorizedOperation\") || (\$.errorCode = \"AccessDenied*\") }"

  metric_transformation {
    name      = "EventCount"
    namespace = "CloudTrailMetrics"
    value     = "1"
  }
}
