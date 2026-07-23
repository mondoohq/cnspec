# Non-compliant: a counted API allows wildcard origins with credentials.
resource "aws_apigatewayv2_api" "fail_count" {
  count         = 2
  name          = "example-api-${count.index}"
  protocol_type = "HTTP"

  cors_configuration {
    allow_credentials = true
    allow_origins     = ["*"]
  }
}
