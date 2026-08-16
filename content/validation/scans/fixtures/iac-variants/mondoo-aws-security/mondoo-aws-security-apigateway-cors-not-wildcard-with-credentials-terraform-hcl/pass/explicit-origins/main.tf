# Compliant: credentials are allowed but origins are explicit, not wildcard.
resource "aws_apigatewayv2_api" "pass_example" {
  name          = "example-api"
  protocol_type = "HTTP"

  cors_configuration {
    allow_credentials = true
    allow_origins     = ["https://app.example.com"]
  }
}
