# Non-compliant: identity_provider_type unset (defaults to SERVICE_MANAGED).
resource "aws_transfer_server" "default_managed" {
  endpoint_type = "PUBLIC"
}
