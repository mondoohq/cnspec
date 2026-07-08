# Compliant: among several metric filters, one watches for failed console
# authentication attempts.
resource "aws_cloudwatch_log_metric_filter" "network_gateway_changes" {
  name           = "network-gateway-changes"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = CreateCustomerGateway) || ($.eventName = DeleteCustomerGateway) || ($.eventName = AttachInternetGateway) || ($.eventName = CreateInternetGateway) || ($.eventName = DeleteInternetGateway) || ($.eventName = DetachInternetGateway) }"

  metric_transformation {
    name      = "NetworkGatewayChanges"
    namespace = "CISBenchmark"
    value     = "1"
  }
}

resource "aws_cloudwatch_log_metric_filter" "console_auth_failures" {
  name           = "console-auth-failures"
  log_group_name = "cloudtrail-logs"
  pattern        = "{ ($.eventName = ConsoleLogin) && ($.errorMessage = \"Failed authentication\") }"

  metric_transformation {
    name      = "ConsoleAuthFailures"
    namespace = "CISBenchmark"
    value     = "1"
  }
}
