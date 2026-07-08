resource "aws_iam_policy" "read_config" {
  name = "read-config"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "s3:GetObject"
        Resource = "arn:aws:s3:::example-bucket/config"
      }
    ]
  })
}
