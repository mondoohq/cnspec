# Compliant: SNS subscription uses the HTTPS protocol.
resource "aws_sns_topic" "example" {
  name = "example-topic"
}

resource "aws_sns_topic_subscription" "example" {
  topic_arn = aws_sns_topic.example.arn
  protocol  = "https"
  endpoint  = "https://example.com/notify"
}
