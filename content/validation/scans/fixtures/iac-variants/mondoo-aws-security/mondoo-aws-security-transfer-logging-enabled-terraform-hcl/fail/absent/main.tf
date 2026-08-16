resource "aws_transfer_server" "example" {
  endpoint_type          = "PUBLIC"
  identity_provider_type = "SERVICE_MANAGED"
}
