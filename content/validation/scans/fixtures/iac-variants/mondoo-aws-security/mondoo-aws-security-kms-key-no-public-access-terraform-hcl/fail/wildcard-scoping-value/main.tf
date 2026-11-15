# Non-compliant: a "*" among the condition values matches any account.
resource "aws_kms_key" "example" {
  description = "application data key"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "WildcardDecrypt"
        Effect    = "Allow"
        Principal = { AWS = "*" }
        Action    = "kms:Decrypt"
        Resource  = "*"
        Condition = {
          "StringLike" = {
            "aws:PrincipalAccount" = ["123456789012", "*"]
          }
        }
      }
    ]
  })
}
