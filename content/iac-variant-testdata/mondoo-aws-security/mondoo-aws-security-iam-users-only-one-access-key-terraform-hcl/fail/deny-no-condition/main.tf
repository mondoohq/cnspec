# Non-compliant: denies CreateAccessKey outright but has no NumericGreaterThan
# condition on iam:AccessKeysCount, so it does not implement the "one key" limit.
resource "aws_iam_policy" "fail_example" {
  name = "deny-create-access-key-unconditional"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "DenyAllKeyCreation"
        Effect   = "Deny"
        Action   = "iam:CreateAccessKey"
        Resource = "*"
      }
    ]
  })
}
