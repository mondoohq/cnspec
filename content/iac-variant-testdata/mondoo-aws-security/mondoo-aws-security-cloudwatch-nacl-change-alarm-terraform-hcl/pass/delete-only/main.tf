# Compliant: metric filter matches network ACL deletions only (second OR branch).
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "nacl-delete-changes"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = DeleteNetworkAcl) || ($.eventName = DeleteNetworkAclEntry) }"

  metric_transformation {
    name      = "NetworkAclDeleteCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
