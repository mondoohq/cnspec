# Non-compliant: SNS topic pinned to signature version 1 (SigV1).
resource "aws_sns_topic" "fail_example" {
  name              = "example-topic"
  signature_version = 1
}
