# Non-compliant: the MFA deny is scoped to a single bucket instead of Resource
# "*", so it does not enforce MFA account-wide as the check requires.
resource "aws_iam_policy" "partial_mfa" {
  name = "partial-mfa"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "DenyBucketWhenNoMFA"
        Effect   = "Deny"
        Action   = "s3:*"
        Resource = "arn:aws:s3:::sensitive-bucket/*"
        Condition = {
          Bool = {
            "aws:MultiFactorAuthPresent" = "false"
          }
        }
      }
    ]
  })
}
