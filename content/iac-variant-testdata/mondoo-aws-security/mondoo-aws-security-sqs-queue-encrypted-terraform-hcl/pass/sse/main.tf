# Compliant: queue enables SQS-managed server-side encryption.
resource "aws_sqs_queue" "pass_example" {
  name                    = "example-queue"
  sqs_managed_sse_enabled = true
}
