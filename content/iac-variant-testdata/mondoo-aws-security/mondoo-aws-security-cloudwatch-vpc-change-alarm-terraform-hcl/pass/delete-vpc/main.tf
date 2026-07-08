# Compliant: CIS VPC pattern including DeleteVpc.
resource "aws_cloudwatch_log_metric_filter" "vpc_changes" {
  name           = "vpc_changes"
  log_group_name = "example-log-group"
  pattern        = "{ (\$.eventName = CreateVpc) || (\$.eventName = DeleteVpc) || (\$.eventName = ModifyVpcAttribute) }"

  metric_transformation {
    name      = "EventCount"
    namespace = "CloudTrailMetrics"
    value     = "1"
  }
}
