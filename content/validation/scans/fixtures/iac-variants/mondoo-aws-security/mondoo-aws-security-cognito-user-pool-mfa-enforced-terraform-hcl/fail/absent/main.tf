# Non-compliant: mfa_configuration is omitted, so MFA defaults to OFF.
resource "aws_cognito_user_pool" "fail_absent" {
  name = "fail-absent"

  password_policy {
    minimum_length = 12
  }
}
