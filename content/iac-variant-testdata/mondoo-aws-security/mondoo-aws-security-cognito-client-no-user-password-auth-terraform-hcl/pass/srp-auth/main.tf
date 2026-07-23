# Compliant: client does not allow user password auth flows.
resource "aws_cognito_user_pool_client" "pass_example" {
  name                = "example"
  user_pool_id        = "us-east-1_example"
  explicit_auth_flows = ["ALLOW_USER_SRP_AUTH", "ALLOW_REFRESH_TOKEN_AUTH"]
}
