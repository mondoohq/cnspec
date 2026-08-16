# Compliant: the redrive policy references the dead-letter queue resource, and the
# dead-letter queue declares itself a redrive target so it needs no DLQ of its own.
resource "aws_sqs_queue" "dlq" {
  name = "example-dlq"

  redrive_allow_policy = jsonencode({
    redrivePermission = "byQueue"
    sourceQueueArns   = [aws_sqs_queue.pass_example.arn]
  })
}

resource "aws_sqs_queue" "pass_example" {
  name = "example-queue"

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = 5
  })
}
