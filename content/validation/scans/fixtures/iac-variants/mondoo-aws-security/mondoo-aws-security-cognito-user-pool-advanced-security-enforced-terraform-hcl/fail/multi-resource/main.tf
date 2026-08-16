# Non-compliant: one of two user pools does not enforce advanced security.
resource "aws_cognito_user_pool" "ok" {
  name = "ok"
  user_pool_add_ons {
    advanced_security_mode = "ENFORCED"
  }
}

resource "aws_cognito_user_pool" "bad" {
  name = "bad"
  user_pool_add_ons {
    advanced_security_mode = "AUDIT"
  }
}
