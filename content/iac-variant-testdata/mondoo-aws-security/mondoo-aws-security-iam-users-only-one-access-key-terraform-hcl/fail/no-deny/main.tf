# Non-compliant: policy allows creating access keys with no limit on the count.
resource "aws_iam_policy" "fail_example" {
  name = "allow-create-access-key"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "iam:CreateAccessKey"
        Resource = "*"
      }
    ]
  })
}
