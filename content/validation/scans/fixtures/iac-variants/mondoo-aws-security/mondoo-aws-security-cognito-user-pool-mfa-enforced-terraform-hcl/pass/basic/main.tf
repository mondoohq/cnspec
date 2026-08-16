# Compliant: MFA is enforced (ON) on the user pool.
resource "aws_cognito_user_pool" "pass_example" {
  name              = "pass-pool"
  mfa_configuration = "ON"
}
