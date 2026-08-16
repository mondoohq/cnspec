# Non-compliant: SNS topic has no KMS master key, so encryption at rest is disabled.
resource "aws_sns_topic" "example" {
  name = "example-topic"
}
