# Non-compliant: a counted user pool leaves advanced security off.
resource "aws_cognito_user_pool" "counted" {
  count = 2
  name  = "example-${count.index}"
  user_pool_add_ons {
    advanced_security_mode = "OFF"
  }
}
