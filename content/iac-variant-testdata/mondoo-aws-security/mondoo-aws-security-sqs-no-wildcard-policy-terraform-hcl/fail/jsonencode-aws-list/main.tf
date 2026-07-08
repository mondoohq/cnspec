# Non-compliant: Principal.AWS is a list containing the "*" wildcard. The structured
# branch must catch this via .contains("*"), matching the native check.
resource "aws_sqs_queue_policy" "fail_example" {
  queue_url = "https://sqs.us-east-1.amazonaws.com/111122223333/example-queue"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicViaAwsList"
        Effect    = "Allow"
        Principal = { AWS = ["*"] }
        Action    = "sqs:SendMessage"
        Resource  = "arn:aws:sqs:us-east-1:111122223333:example-queue"
      }
    ]
  })
}
