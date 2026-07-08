# Compliant: wildcard origin is allowed but credentials are not.
resource "aws_apigatewayv2_api" "pass_example" {
  name          = "example-api"
  protocol_type = "HTTP"

  cors_configuration {
    allow_credentials = false
    allow_origins     = ["*"]
  }
}
