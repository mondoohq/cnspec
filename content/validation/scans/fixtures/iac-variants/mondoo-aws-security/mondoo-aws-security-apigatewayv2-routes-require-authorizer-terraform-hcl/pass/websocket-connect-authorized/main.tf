# Compliant: the WebSocket API authorizes $connect. API Gateway rejects
# authorization on any other WebSocket route, so $default is not checked.
resource "aws_apigatewayv2_api" "ws" {
  name                       = "example-ws"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}

resource "aws_apigatewayv2_route" "connect" {
  api_id             = aws_apigatewayv2_api.ws.id
  route_key          = "$connect"
  authorization_type = "AWS_IAM"
}

resource "aws_apigatewayv2_route" "default" {
  api_id    = aws_apigatewayv2_api.ws.id
  route_key = "$default"
}
