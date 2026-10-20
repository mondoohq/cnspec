# Non-compliant: without security_policy_name the provider attaches
# TransferSecurityPolicy-2018-11.
resource "aws_transfer_server" "example" {
  endpoint_type          = "PUBLIC"
  identity_provider_type = "SERVICE_MANAGED"
}
