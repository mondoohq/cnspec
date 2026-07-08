# Compliant: metric filter matches network ACL change API calls.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "nacl-changes"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = CreateNetworkAcl) || ($.eventName = DeleteNetworkAcl) || ($.eventName = CreateNetworkAclEntry) }"

  metric_transformation {
    name      = "NetworkAclEventCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
