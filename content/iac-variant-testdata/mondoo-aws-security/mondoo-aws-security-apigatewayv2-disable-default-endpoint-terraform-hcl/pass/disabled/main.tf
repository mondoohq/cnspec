resource "aws_apigatewayv2_api" "events" {
  name                         = "events-api"
  protocol_type                = "HTTP"
  disable_execute_api_endpoint = true
}
