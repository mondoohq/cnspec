# Compliant: among several metric filters, one watches for CMK disable and
# scheduled deletion.
resource "aws_cloudwatch_log_metric_filter" "root_usage" {
  name           = "root-account-usage"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ $.userIdentity.type = \"Root\" && $.userIdentity.invokedBy NOT EXISTS && $.eventType != \"AwsServiceEvent\" }"

  metric_transformation {
    name      = "RootAccountUsage"
    namespace = "CISBenchmark"
    value     = "1"
  }
}

resource "aws_cloudwatch_log_metric_filter" "cmk_disable_delete" {
  name           = "cmk-disable-delete"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventSource = kms.amazonaws.com) && (($.eventName = DisableKey) || ($.eventName = ScheduleKeyDeletion)) }"

  metric_transformation {
    name      = "CMKDisableOrDelete"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
