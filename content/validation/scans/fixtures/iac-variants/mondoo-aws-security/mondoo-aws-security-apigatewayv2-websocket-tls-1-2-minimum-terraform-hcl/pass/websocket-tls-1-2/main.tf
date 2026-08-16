# Compliant: WebSocket API with a custom domain enforcing TLS 1.2.
resource "aws_apigatewayv2_api" "pass_example" {
  name          = "example-ws"
  protocol_type = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}

resource "aws_apigatewayv2_domain_name" "pass_example" {
  domain_name = "ws.example.com"

  domain_name_configuration {
    certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}
