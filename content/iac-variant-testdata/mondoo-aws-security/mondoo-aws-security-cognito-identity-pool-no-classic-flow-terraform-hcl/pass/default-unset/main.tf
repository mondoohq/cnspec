# Compliant: allow_classic_flow is not set and defaults to false.
resource "aws_cognito_identity_pool" "pass_default" {
  identity_pool_name               = "example"
  allow_unauthenticated_identities = false
}
