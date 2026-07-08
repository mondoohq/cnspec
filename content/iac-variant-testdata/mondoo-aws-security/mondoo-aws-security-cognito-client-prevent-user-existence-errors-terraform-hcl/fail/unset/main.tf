# Non-compliant: prevent_user_existence_errors is not set, leaving legacy behavior.
resource "aws_cognito_user_pool_client" "fail_unset" {
  name         = "example"
  user_pool_id = aws_cognito_user_pool.example.id
}
