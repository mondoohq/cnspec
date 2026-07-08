# Non-compliant: advanced security is turned off, not enforced.
resource "aws_cognito_user_pool" "fail_off" {
  name = "example"

  user_pool_add_ons {
    advanced_security_mode = "OFF"
  }
}
