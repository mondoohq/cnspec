# Non-compliant: the dead-letter queue neither carries a redrive policy nor declares
# itself a redrive target, so it is indistinguishable from an unprotected queue.
resource "aws_sqs_queue" "dlq" {
  name = "example-dlq"
}

resource "aws_sqs_queue" "example" {
  name = "example-queue"

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = 5
  })
}
