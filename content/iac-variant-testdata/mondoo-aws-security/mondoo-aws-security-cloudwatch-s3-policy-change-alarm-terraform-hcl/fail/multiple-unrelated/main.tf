# Non-compliant: several metric filters exist, none tracking S3 bucket policy changes.
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
