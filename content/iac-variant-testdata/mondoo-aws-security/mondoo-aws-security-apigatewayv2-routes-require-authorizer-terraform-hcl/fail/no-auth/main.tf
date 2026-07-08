# Non-compliant: non-OPTIONS route has no authorizer.
resource "aws_apigatewayv2_route" "fail_example" {
  api_id             = "abc123"
  route_key          = "GET /items"
  authorization_type = "NONE"
}
