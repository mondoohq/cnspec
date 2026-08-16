# Non-compliant: several metric filters exist, none tracking unauthorized API calls.
resource "aws_cloudwatch_log_metric_filter" "console_signin_failures" {
  name           = "console-signin-failures"
  log_group_name = "example-log-group"
  pattern        = "{ ($.eventName = ConsoleLogin) && ($.errorMessage = \"Failed authentication\") }"

  metric_transformation {
    name      = "ConsoleSigninFailureCount"
    namespace = "CloudTrailMetrics"
    value     = "1"
  }
}

resource "aws_cloudwatch_log_metric_filter" "iam_policy_changes" {
  name           = "iam-policy-changes"
  log_group_name = "example-log-group"
  pattern        = "{ ($.eventName = DeleteGroupPolicy) || ($.eventName = PutGroupPolicy) }"

  metric_transformation {
    name      = "IAMPolicyEventCount"
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
