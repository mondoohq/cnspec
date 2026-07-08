# Non-compliant: SNS topic does not set signature_version (defaults to SigV1).
resource "aws_sns_topic" "fail_example" {
  name = "example-topic"
}
