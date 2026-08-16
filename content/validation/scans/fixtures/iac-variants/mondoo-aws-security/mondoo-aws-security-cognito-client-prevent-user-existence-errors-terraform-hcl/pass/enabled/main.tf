# Compliant: user existence errors are prevented.
resource "aws_cognito_user_pool_client" "pass_example" {
  name                          = "example"
  user_pool_id                  = "us-east-1_example"
  prevent_user_existence_errors = "ENABLED"
}
