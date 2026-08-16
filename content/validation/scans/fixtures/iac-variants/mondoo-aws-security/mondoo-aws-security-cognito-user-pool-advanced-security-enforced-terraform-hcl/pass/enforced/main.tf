# Compliant: advanced security features are enforced.
resource "aws_cognito_user_pool" "pass_example" {
  name = "example"

  user_pool_add_ons {
    advanced_security_mode = "ENFORCED"
  }
}
