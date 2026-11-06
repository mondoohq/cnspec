# Encrypted under a customer managed key.
resource "aws_bedrockagentcore_policy_engine" "example" {
  name = "tools_engine"
  encryption_key_arn = "arn:aws:kms:us-east-1:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"
}
