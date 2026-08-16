# Non-compliant: advanced security is only in audit mode, not enforced.
resource "aws_cognito_user_pool" "fail_example" {
  name = "example"

  user_pool_add_ons {
    advanced_security_mode = "AUDIT"
  }
}
