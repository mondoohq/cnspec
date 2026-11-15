# Non-compliant: an HTTP API $default route without authorization. The
# WebSocket API in the same configuration does not exempt it.
resource "aws_apigatewayv2_api" "ws" {
  name                       = "example-ws"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}

resource "aws_apigatewayv2_api" "http" {
  name          = "example-http"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_route" "connect" {
  api_id             = aws_apigatewayv2_api.ws.id
  route_key          = "$connect"
  authorization_type = "AWS_IAM"
}

resource "aws_apigatewayv2_route" "http_default" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "$default"
}
