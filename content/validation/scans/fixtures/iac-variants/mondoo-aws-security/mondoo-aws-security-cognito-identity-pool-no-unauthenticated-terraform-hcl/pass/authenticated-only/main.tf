# Compliant: unauthenticated (guest) identities are not allowed.
resource "aws_cognito_identity_pool" "pass_example" {
  identity_pool_name               = "example"
  allow_unauthenticated_identities = false
}
