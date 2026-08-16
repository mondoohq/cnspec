# Non-compliant: client allows plaintext user password auth flow.
resource "aws_cognito_user_pool_client" "fail_example" {
  name                = "example"
  user_pool_id        = "us-east-1_example"
  explicit_auth_flows = ["ALLOW_USER_PASSWORD_AUTH", "ALLOW_REFRESH_TOKEN_AUTH"]
}
