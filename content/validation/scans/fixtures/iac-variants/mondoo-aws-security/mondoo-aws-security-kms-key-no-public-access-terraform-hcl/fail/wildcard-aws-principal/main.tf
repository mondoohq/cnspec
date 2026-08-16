# Non-compliant: KMS key policy allows any AWS account via Principal.AWS = "*".
resource "aws_kms_key" "fail_example" {
  description = "public key"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowAllAws"
        Effect    = "Allow"
        Principal = { AWS = "*" }
        Action    = "kms:Decrypt"
        Resource  = "*"
      }
    ]
  })
}
