# Non-compliant: queue sets no KMS key (no CMK encryption).
resource "aws_sqs_queue" "fail_example" {
  name = "example-queue"
}
