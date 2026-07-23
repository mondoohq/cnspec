# Non-compliant: uses the default SERVICE_MANAGED identity provider.
resource "aws_transfer_server" "managed" {
  identity_provider_type = "SERVICE_MANAGED"
}
