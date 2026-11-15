# Compliant: wildcard principal scoped by kms:CallerAccount, the shape of an AWS managed key policy.
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
          "StringEquals" = {
            "kms:CallerAccount" = "123456789012"
            "kms:ViaService" = "ec2.us-east-1.amazonaws.com"
          }
        }
      }
    ]
  })
}
