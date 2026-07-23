resource "aws_iam_policy" "read_write" {
  name = "read-write"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
        ]
        Resource = [
          "arn:aws:s3:::example-bucket/data",
          "arn:aws:s3:::example-bucket/config",
        ]
      }
    ]
  })
}
