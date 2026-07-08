# Non-compliant: WebSocket API where one of two domains allows TLS 1.0.
resource "aws_apigatewayv2_api" "fail_ws" {
  name                       = "example-ws"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}

resource "aws_apigatewayv2_domain_name" "ok" {
  domain_name = "ws-secure.example.com"

  domain_name_configuration {
    certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}

resource "aws_apigatewayv2_domain_name" "bad" {
  domain_name = "ws-legacy.example.com"

  domain_name_configuration {
    certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_0"
  }
}
