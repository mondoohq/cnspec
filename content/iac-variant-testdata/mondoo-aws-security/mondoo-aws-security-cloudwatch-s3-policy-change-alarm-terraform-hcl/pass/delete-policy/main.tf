# Compliant: pattern including DeleteBucketPolicy.
resource "aws_cloudwatch_log_metric_filter" "s3_policy_changes" {
  name           = "s3_policy_changes"
  log_group_name = "example-log-group"
  pattern        = "{ (\$.eventName = PutBucketAcl) || (\$.eventName = DeleteBucketPolicy) || (\$.eventName = PutBucketAcl) }"

  metric_transformation {
    name      = "EventCount"
    namespace = "CloudTrailMetrics"
    value     = "1"
  }
}
