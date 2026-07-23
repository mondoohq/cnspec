# Non-compliant: queue sets neither SSE-SQS nor a KMS key.
resource "aws_sqs_queue" "fail_example" {
  name = "example-queue"
}
