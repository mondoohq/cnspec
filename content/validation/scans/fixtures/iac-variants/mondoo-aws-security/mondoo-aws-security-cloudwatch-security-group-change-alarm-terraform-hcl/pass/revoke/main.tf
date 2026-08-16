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

# CloudTrail delivering to CloudWatch Logs; the CIS monitoring checks only apply
# when a trail feeds a metric filter, so the terraform-hcl filter requires both.
resource "aws_cloudtrail" "example" {
  name                       = "example-trail"
  s3_bucket_name             = "example-cloudtrail-logs"
  cloud_watch_logs_group_arn = "arn:aws:logs:us-east-1:123456789012:log-group:example:*"
}
