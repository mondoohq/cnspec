# Compliant: CIS route-table pattern including DeleteRoute.
resource "aws_cloudwatch_log_metric_filter" "route_table_changes" {
  name           = "route_table_changes"
  log_group_name = "example-log-group"
  pattern        = "{ (\$.eventName = CreateRouteTable) || (\$.eventName = ReplaceRoute) || (\$.eventName = DeleteRoute) || (\$.eventName = DeleteRouteTable) }"

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
