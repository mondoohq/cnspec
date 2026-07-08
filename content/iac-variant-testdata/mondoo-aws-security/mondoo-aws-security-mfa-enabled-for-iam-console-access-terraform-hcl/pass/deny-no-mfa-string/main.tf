# Compliant: IAM policy denies all actions when MFA is not present.
# Condition value written as the string "false", the form AWS documents for
# aws:MultiFactorAuthPresent.
resource "aws_iam_policy" "require_mfa" {
  name = "require-mfa"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "DenyAllExceptWhenMFAPresent"
        Effect   = "Deny"
        Action   = "*"
        Resource = "*"
        Condition = {
          Bool = {
            "aws:MultiFactorAuthPresent" = "false"
          }
        }
      }
    ]
  })
}
