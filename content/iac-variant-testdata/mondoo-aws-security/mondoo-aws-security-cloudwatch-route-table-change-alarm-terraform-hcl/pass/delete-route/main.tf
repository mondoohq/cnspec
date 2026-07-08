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
