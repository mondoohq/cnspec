# Non-compliant: client enables the legacy USER_PASSWORD_AUTH flow.
resource "aws_cognito_user_pool_client" "fail_legacy" {
  name         = "example"
  user_pool_id = aws_cognito_user_pool.example.id

  explicit_auth_flows = [
    "USER_PASSWORD_AUTH",
    "REFRESH_TOKEN_AUTH",
  ]
}
