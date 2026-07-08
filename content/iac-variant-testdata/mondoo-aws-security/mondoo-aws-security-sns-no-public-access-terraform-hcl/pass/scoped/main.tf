# Compliant: SNS topic policy grants access only to a specific account principal.
resource "aws_sns_topic" "example" {
  name = "example-topic"
}

resource "aws_sns_topic_policy" "example" {
  arn = "arn:aws:sns:us-east-1:111122223333:example-topic"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowOwnAccount"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::111122223333:root" }
        Action    = "SNS:Publish"
        Resource  = "arn:aws:sns:us-east-1:111122223333:example-topic"
      }
    ]
  })
}
