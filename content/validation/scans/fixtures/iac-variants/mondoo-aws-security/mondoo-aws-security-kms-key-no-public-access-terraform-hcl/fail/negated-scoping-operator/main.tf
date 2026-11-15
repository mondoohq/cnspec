# Non-compliant: StringNotEquals on aws:PrincipalOrgID allows every principal outside the organization.
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
          "StringNotEquals" = {
            "aws:PrincipalOrgID" = "o-a1b2c3d4e5"
          }
        }
      }
    ]
  })
}
