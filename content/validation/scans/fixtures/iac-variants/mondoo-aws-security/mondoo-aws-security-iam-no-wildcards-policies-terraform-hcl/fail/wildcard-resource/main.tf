resource "aws_iam_policy" "any_resource" {
  name = "any-resource"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "s3:GetObject"
        Resource = "*"
      }
    ]
  })
}
