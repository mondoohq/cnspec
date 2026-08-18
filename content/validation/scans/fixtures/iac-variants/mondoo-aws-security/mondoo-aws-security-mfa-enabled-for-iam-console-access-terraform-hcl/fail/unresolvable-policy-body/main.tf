# Non-compliant, and the regression case for a false pass that shipped:
# this is fail/allow-only with one change, `Resource` points at another
# resource instead of a literal. That makes jsonencode() collapse to an
# empty list, so the policy body cannot be read. No MFA is enforced, and
# an unreadable body must not be counted as evidence that it is.
resource "aws_s3_bucket" "data" {
  bucket = "example-data"
}

resource "aws_iam_policy" "allow_read" {
  name = "allow-read"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "AllowRead"
        Effect   = "Allow"
        Action   = "s3:GetObject"
        Resource = aws_s3_bucket.data.arn
      }
    ]
  })
}

# A console user exists, so the MFA-enforcement check applies.
resource "aws_iam_user_login_profile" "example" {
  user = "example-user"
}
