# Non-compliant: one of two APIs allows wildcard origins with credentials.
resource "aws_apigatewayv2_api" "ok" {
  name          = "safe-api"
  protocol_type = "HTTP"

  cors_configuration {
    allow_credentials = true
    allow_origins     = ["https://app.example.com"]
  }
}

resource "aws_apigatewayv2_api" "bad" {
  name          = "unsafe-api"
  protocol_type = "HTTP"

  cors_configuration {
    allow_credentials = true
    allow_origins     = ["*"]
  }
}
