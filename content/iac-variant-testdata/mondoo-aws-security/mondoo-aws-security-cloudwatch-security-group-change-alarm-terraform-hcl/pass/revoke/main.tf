# Compliant: pattern including RevokeSecurityGroupIngress.
resource "aws_cloudwatch_log_metric_filter" "sg_changes" {
  name           = "sg_changes"
  log_group_name = "example-log-group"
  pattern        = "{ (\$.eventName = AuthorizeSecurityGroupEgress) || (\$.eventName = RevokeSecurityGroupIngress) || (\$.eventName = CreateSecurityGroup) }"

  metric_transformation {
    name      = "EventCount"
    namespace = "CloudTrailMetrics"
    value     = "1"
  }
}
