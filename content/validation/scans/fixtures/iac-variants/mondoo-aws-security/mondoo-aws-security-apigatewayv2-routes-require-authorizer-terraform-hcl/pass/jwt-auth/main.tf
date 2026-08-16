# Compliant: non-OPTIONS route requires an authorizer.
resource "aws_apigatewayv2_route" "pass_example" {
  api_id             = "abc123"
  route_key          = "GET /items"
  authorization_type = "JWT"
  authorizer_id      = "auth123"
}
