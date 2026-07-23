# Compliant: queue configures a redrive policy pointing at a dead-letter queue.
resource "aws_sqs_queue" "pass_example" {
  name = "example-queue"

  redrive_policy = <<POLICY
{
  "deadLetterTargetArn": "arn:aws:sqs:us-east-1:111122223333:example-dlq",
  "maxReceiveCount": 5
}
POLICY
}
