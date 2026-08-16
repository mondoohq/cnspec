resource "aws_iam_user" "this" {
  name = "alice"
}

resource "aws_iam_user_policy" "this" {
  name = "inline"
  user = aws_iam_user.this.name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "s3:GetObject"
        Resource = "arn:aws:s3:::example-bucket/data"
      }
    ]
  })
}
