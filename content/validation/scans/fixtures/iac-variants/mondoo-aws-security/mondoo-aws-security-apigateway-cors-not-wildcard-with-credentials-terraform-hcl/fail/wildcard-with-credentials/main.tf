# Non-compliant: wildcard origin combined with credentials allowed.
resource "aws_apigatewayv2_api" "fail_example" {
  name          = "example-api"
  protocol_type = "HTTP"

  cors_configuration {
    allow_credentials = true
    allow_origins     = ["*"]
  }
}
