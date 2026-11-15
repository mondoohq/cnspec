# Non-compliant: aws:SecureTransport does not limit who can call, so the wildcard principal stays public.
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
          "Bool" = {
            "aws:SecureTransport" = "true"
          }
        }
      }
    ]
  })
}
