# Non-compliant: SNS topic policy has no statement denying insecure (non-TLS) transport.
resource "aws_sns_topic" "example" {
  name = "example-topic"
}

resource "aws_sns_topic_policy" "example" {
  arn = aws_sns_topic.example.arn

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowOwnAccount"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::111122223333:root" }
        Action    = "SNS:Publish"
        Resource  = aws_sns_topic.example.arn
      }
    ]
  })
}
