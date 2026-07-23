# Non-compliant: IAM policy only grants access and has no MFA-enforcing deny
# statement, so nothing forces console users to authenticate with MFA.
resource "aws_iam_policy" "allow_read" {
  name = "allow-read"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "AllowRead"
        Effect   = "Allow"
        Action   = "s3:GetObject"
        Resource = "*"
      }
    ]
  })
}

# A console user exists, so the MFA-enforcement check applies.
resource "aws_iam_user_login_profile" "example" {
  user = "example-user"
}
