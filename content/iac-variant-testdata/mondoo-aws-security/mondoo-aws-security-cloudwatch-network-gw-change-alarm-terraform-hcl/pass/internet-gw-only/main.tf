# Compliant: metric filter matches internet gateway attachment only (second OR branch).
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "internet-gw-changes"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = AttachInternetGateway) || ($.eventName = DetachInternetGateway) }"

  metric_transformation {
    name      = "InternetGatewayEventCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
