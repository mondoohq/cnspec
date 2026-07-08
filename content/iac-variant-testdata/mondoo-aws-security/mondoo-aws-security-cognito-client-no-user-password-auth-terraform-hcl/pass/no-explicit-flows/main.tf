# Compliant: explicit_auth_flows is not set, so no password auth flow is enabled.
resource "aws_cognito_user_pool_client" "pass_default" {
  name         = "example"
  user_pool_id = aws_cognito_user_pool.example.id
}
