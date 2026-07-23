# Non-compliant: MFA is optional, not enforced.
resource "aws_cognito_user_pool" "fail_example" {
  name              = "fail-pool"
  mfa_configuration = "OPTIONAL"
}
