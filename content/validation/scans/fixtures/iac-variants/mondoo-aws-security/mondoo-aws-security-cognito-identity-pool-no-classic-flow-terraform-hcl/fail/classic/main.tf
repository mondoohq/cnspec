# Non-compliant: classic (basic) authentication flow is enabled.
resource "aws_cognito_identity_pool" "fail_example" {
  identity_pool_name               = "example"
  allow_unauthenticated_identities = false
  allow_classic_flow               = true
}
