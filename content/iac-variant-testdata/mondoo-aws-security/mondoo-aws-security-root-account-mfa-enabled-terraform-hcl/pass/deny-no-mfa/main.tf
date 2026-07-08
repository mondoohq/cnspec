# Compliant: IAM policy denies access when MFA is not present.
resource "aws_iam_policy" "pass_example" {
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
          BoolIfExists = {
            "aws:MultiFactorAuthPresent" = "false"
          }
        }
      }
    ]
  })
}
