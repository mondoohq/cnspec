# A Deny statement may legitimately use wildcards to block everything by default.
resource "aws_iam_policy" "deny_all" {
  name = "deny-all"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Deny"
        Action   = "*"
        Resource = "*"
      }
    ]
  })
}
