resource "aws_secretsmanager_secret_policy" "this" {
  secret_arn = "arn:aws:secretsmanager:us-east-1:123456789012:secret:example"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "ScopedRead"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::123456789012:root" }
        Action    = "secretsmanager:GetSecretValue"
        Resource  = "*"
      }
    ]
  })
}
