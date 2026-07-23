# Compliant: SNS topic uses signature version 2 (SigV4).
resource "aws_sns_topic" "pass_example" {
  name              = "example-topic"
  signature_version = 2
}
