# Compliant: SNS subscription delivers to SQS, which is not the insecure http protocol.
resource "aws_sns_topic" "example" {
  name = "example-topic"
}

resource "aws_sqs_queue" "example" {
  name = "example-queue"
}

resource "aws_sns_topic_subscription" "example" {
  topic_arn = aws_sns_topic.example.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.example.arn
}
