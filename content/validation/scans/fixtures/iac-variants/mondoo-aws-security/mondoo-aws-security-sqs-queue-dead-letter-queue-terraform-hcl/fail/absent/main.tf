# Non-compliant: queue has no redrive policy, so no dead-letter queue.
resource "aws_sqs_queue" "fail_example" {
  name = "example-queue"
}
