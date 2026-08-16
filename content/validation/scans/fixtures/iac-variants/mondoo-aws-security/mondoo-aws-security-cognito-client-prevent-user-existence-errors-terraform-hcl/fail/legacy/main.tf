# Non-compliant: legacy behavior reveals whether a user exists.
resource "aws_cognito_user_pool_client" "fail_example" {
  name                          = "example"
  user_pool_id                  = "us-east-1_example"
  prevent_user_existence_errors = "LEGACY"
}
