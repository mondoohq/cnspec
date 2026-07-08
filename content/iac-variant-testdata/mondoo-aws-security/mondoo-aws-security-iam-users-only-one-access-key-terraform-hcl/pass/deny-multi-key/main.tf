# Compliant: policy denies creating a second access key per user.
resource "aws_iam_policy" "pass_example" {
  name = "deny-multiple-access-keys"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Deny"
        Action = "iam:CreateAccessKey"
        Condition = {
          NumericGreaterThan = {
            "iam:AccessKeysCount" = 1
          }
        }
      }
    ]
  })
}
