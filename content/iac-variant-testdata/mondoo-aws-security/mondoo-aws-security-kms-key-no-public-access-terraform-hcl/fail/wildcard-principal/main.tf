# Non-compliant: KMS key policy allows the wildcard principal "*".
resource "aws_kms_key" "fail_example" {
  description = "public key"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowPublic"
        Effect    = "Allow"
        Principal = "*"
        Action    = "kms:*"
        Resource  = "*"
      }
    ]
  })
}
