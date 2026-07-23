resource "aws_s3_bucket_policy" "example" {
  bucket = "my-bucket"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowRead"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::123456789012:root" }
        Action    = "s3:GetObject"
        Resource  = "arn:aws:s3:::my-bucket/*"
      },
      {
        Sid       = "DenyInsecureTransport"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource  = "arn:aws:s3:::my-bucket/*"
        Condition = { Bool = { "aws:SecureTransport" = false } }
      }
    ]
  })
}
