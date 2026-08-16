# Non-compliant: SNS topic has no KMS key configured, so messages are unencrypted at rest.
resource "aws_sns_topic" "example" {
  name = "example-topic"
}
