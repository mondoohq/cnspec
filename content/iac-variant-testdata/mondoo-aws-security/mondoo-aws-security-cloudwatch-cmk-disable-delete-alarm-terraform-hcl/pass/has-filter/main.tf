# Compliant: metric filter watches for CMK disable and scheduled deletion.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "cmk-disable-delete"
  log_group_name = "example"
  pattern        = "{ ($.eventSource = kms.amazonaws.com) && (($.eventName = DisableKey) || ($.eventName = ScheduleKeyDeletion)) }"

  metric_transformation {
    name      = "CMKDisableOrDelete"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
