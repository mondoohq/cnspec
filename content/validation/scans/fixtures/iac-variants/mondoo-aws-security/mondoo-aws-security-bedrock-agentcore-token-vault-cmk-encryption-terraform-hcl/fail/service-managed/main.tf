# The vault is explicitly set to the service-managed key.
resource "aws_bedrockagentcore_token_vault_cmk" "example" {
  kms_configuration {
    key_type = "ServiceManagedKey"
  }
}
