resource "aws_transfer_server" "example" {
  endpoint_type          = "PUBLIC"
  identity_provider_type = "SERVICE_MANAGED"
  security_policy_name   = "TransferSecurityPolicy-2024-01"
}
