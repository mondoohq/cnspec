# The execute-api endpoint stays reachable alongside the custom domain, so WAF
# rules and domain-level controls attached to the custom domain can be bypassed.
resource "aws_api_gateway_rest_api" "orders" {
  name        = "orders-api"
  description = "Order service, fronted by a custom domain"

  endpoint_configuration {
    types = ["REGIONAL"]
  }
}
