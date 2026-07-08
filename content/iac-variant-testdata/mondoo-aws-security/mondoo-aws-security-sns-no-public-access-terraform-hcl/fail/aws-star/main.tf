# Non-compliant: SNS topic policy allows any AWS account (Principal.AWS = "*").
resource "aws_sns_topic" "example" {
  name = "example-topic"
}

resource "aws_sns_topic_policy" "example" {
  arn = aws_sns_topic.example.arn

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AnyAccountPublish"
        Effect    = "Allow"
        Principal = { AWS = "*" }
        Action    = "SNS:Publish"
        Resource  = aws_sns_topic.example.arn
      }
    ]
  })
}
