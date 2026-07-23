# Non-compliant: SNS topic policy allows every principal (Principal = "*").
resource "aws_sns_topic" "example" {
  name = "example-topic"
}

resource "aws_sns_topic_policy" "example" {
  arn = "arn:aws:sns:us-east-1:111122223333:example-topic"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicPublish"
        Effect    = "Allow"
        Principal = "*"
        Action    = "SNS:Publish"
        Resource  = "arn:aws:sns:us-east-1:111122223333:example-topic"
      }
    ]
  })
}
