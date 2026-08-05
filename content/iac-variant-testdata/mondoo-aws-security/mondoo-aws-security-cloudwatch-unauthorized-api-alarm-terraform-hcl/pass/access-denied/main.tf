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

# CloudTrail delivering to CloudWatch Logs; the CIS monitoring checks only apply
# when a trail feeds a metric filter, so the terraform-hcl filter requires both.
resource "aws_cloudtrail" "example" {
  name                       = "example-trail"
  s3_bucket_name             = "example-cloudtrail-logs"
  cloud_watch_logs_group_arn = "arn:aws:logs:us-east-1:123456789012:log-group:example:*"
}

# Alarm on the filter's metric; the CIS control requires both a metric filter
# and an alarm that notifies, so the check asserts an alarm with actions exists.
resource "aws_cloudwatch_metric_alarm" "example" {
  alarm_name          = "example-alarm"
  namespace           = "CISBenchmark"
  metric_name         = "ExampleMetric"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  period              = 300
  statistic           = "Sum"
  threshold           = 1
  alarm_actions       = ["arn:aws:sns:us-east-1:123456789012:example-topic"]
}
