# Non-compliant: IAM policy has no Deny statement enforcing MFA.
resource "aws_iam_policy" "fail_example" {
  name = "no-mfa"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "s3:GetObject"
        Resource = "arn:aws:s3:::example-bucket/*"
      }
    ]
  })
}
