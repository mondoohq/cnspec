# Compliant: queue policy scopes access to a specific account principal.
resource "aws_sqs_queue_policy" "pass_example" {
  queue_url = "https://sqs.us-east-1.amazonaws.com/111122223333/example-queue"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowScoped"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::111122223333:root" }
        Action    = "sqs:SendMessage"
        Resource  = "arn:aws:sqs:us-east-1:111122223333:example-queue"
      }
    ]
  })
}
