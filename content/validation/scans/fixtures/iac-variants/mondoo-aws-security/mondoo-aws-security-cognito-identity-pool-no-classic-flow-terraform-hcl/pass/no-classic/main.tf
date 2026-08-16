# Compliant: classic (basic) authentication flow is disabled.
resource "aws_cognito_identity_pool" "pass_example" {
  identity_pool_name               = "example"
  allow_unauthenticated_identities = false
  allow_classic_flow               = false
}
