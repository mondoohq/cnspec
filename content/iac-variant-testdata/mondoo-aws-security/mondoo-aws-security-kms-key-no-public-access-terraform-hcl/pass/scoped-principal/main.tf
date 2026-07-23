# Compliant: KMS key policy grants access to a scoped principal, not "*".
resource "aws_kms_key" "pass_example" {
  description = "scoped key"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowRoot"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::111122223333:root" }
        Action    = "kms:*"
        Resource  = "*"
      }
    ]
  })
}
