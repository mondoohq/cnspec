# Non-compliant: unauthenticated (guest) identities are allowed.
resource "aws_cognito_identity_pool" "fail_example" {
  identity_pool_name               = "example"
  allow_unauthenticated_identities = true
}
