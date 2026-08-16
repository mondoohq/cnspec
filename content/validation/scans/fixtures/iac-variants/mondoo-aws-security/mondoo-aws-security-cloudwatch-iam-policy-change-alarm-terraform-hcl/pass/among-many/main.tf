# Compliant: among several metric filters, one watches for IAM policy changes.
resource "aws_cloudwatch_log_metric_filter" "route_table_changes" {
  name           = "route-table-changes"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = CreateRoute) || ($.eventName = CreateRouteTable) || ($.eventName = ReplaceRoute) || ($.eventName = DeleteRouteTable) || ($.eventName = DeleteRoute) }"

  metric_transformation {
    name      = "RouteTableChanges"
    namespace = "CISBenchmark"
    value     = "1"
  }
}

resource "aws_cloudwatch_log_metric_filter" "iam_policy_changes" {
  name           = "iam-policy-changes"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = DeleteGroupPolicy) || ($.eventName = DeleteRolePolicy) || ($.eventName = DeleteUserPolicy) || ($.eventName = PutGroupPolicy) || ($.eventName = PutRolePolicy) || ($.eventName = PutUserPolicy) || ($.eventName = CreatePolicy) || ($.eventName = DeletePolicy) || ($.eventName = AttachRolePolicy) || ($.eventName = DetachRolePolicy) }"

  metric_transformation {
    name      = "IAMPolicyChanges"
    namespace = "CISBenchmark"
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
