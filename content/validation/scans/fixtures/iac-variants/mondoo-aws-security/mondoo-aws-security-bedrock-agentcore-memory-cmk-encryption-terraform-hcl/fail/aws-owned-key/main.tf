# Encrypted with the AWS owned default key.
resource "aws_bedrockagentcore_memory" "example" {
  name                  = "support_memory"
  event_expiry_duration = 30
}
