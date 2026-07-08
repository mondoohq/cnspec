# Compliant: metric filter watches for IAM policy changes.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "iam-policy-changes"
  log_group_name = "example"
  pattern        = "{ ($.eventName = CreatePolicy) || ($.eventName = DeletePolicy) || ($.eventName = AttachRolePolicy) }"

  metric_transformation {
    name      = "IAMPolicyChanges"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
