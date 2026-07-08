# Non-compliant: metric filter does not watch for CloudTrail configuration changes.
resource "aws_cloudwatch_log_metric_filter" "fail_example" {
  name           = "unrelated"
  log_group_name = "example"
  pattern        = "{ ($.eventName = SomethingElse) }"

  metric_transformation {
    name      = "Unrelated"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
