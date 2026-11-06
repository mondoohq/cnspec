# Stored OAuth tokens and API keys are encrypted under an account-owned key.
resource "aws_bedrockagentcore_token_vault_cmk" "example" {
  kms_configuration {
    key_type    = "CustomerManagedKey"
    kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"
  }
}
