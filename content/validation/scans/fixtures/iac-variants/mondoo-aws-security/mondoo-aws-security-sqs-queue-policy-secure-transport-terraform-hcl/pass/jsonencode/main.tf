# Compliant: queue policy denies any request not using TLS (aws:SecureTransport false).
resource "aws_sqs_queue_policy" "pass_example" {
  queue_url = "https://sqs.us-east-1.amazonaws.com/111122223333/example-queue"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "DenyInsecureTransport"
        Effect    = "Deny"
        Principal = "*"
        Action    = "sqs:*"
        Resource  = "arn:aws:sqs:us-east-1:111122223333:example-queue"
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      }
    ]
  })
}
