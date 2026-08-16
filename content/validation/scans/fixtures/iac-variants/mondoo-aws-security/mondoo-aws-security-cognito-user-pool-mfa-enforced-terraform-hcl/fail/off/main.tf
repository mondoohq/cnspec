# Non-compliant: MFA is explicitly disabled on the user pool.
resource "aws_cognito_user_pool" "fail_off" {
  name              = "fail-off"
  mfa_configuration = "OFF"
}
