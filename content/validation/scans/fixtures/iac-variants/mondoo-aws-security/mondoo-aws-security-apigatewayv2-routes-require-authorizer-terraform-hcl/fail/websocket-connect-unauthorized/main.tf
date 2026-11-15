# Non-compliant: the WebSocket $connect route has no authorization, so any
# client can open a connection.
resource "aws_apigatewayv2_api" "ws" {
  name                       = "example-ws"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}

resource "aws_apigatewayv2_route" "connect" {
  api_id             = aws_apigatewayv2_api.ws.id
  route_key          = "$connect"
  authorization_type = "NONE"
}

resource "aws_apigatewayv2_route" "default" {
  api_id    = aws_apigatewayv2_api.ws.id
  route_key = "$default"
}
