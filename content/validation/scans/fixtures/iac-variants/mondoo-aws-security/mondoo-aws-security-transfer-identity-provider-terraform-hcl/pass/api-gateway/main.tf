# Compliant: custom identity provider via API Gateway, not SERVICE_MANAGED.
resource "aws_transfer_server" "custom" {
  identity_provider_type = "API_GATEWAY"
  url                    = "https://api.example.com/prod/auth"
  invocation_role        = aws_iam_role.invocation.arn
}
