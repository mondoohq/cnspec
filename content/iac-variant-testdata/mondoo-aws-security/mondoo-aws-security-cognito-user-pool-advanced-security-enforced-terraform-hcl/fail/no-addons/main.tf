# Non-compliant: no user_pool_add_ons block, so advanced security is not enforced.
resource "aws_cognito_user_pool" "fail_no_addons" {
  name = "example"
}
