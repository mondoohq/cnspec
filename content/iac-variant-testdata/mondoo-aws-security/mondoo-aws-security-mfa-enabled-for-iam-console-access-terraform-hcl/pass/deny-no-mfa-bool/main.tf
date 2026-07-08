# Compliant: IAM policy denies all actions when MFA is not present.
# Condition value written as a bare HCL boolean false, which jsonencode emits as
# the JSON literal false.
resource "aws_iam_policy" "require_mfa" {
  name = "require-mfa"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "DenyWhenNoMFA"
        Effect   = "Deny"
        Action   = "*"
        Resource = "*"
        Condition = {
          Bool = {
            "aws:MultiFactorAuthPresent" = false
          }
        }
      }
    ]
  })
}
