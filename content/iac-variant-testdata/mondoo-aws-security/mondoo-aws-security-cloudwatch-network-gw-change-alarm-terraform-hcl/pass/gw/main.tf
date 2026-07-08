# Compliant: metric filter matches network gateway change API calls.
resource "aws_cloudwatch_log_metric_filter" "pass_example" {
  name           = "network-gw-changes"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = CreateCustomerGateway) || ($.eventName = DeleteCustomerGateway) || ($.eventName = AttachInternetGateway) }"

  metric_transformation {
    name      = "GatewayEventCount"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
