resource "aws_s3_bucket_policy" "example" {
  bucket = "my-bucket"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowAccountRoot"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::123456789012:root" }
        Action    = "s3:GetObject"
        Resource  = "arn:aws:s3:::my-bucket/*"
      },
      {
        Sid       = "PublicRead"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "arn:aws:s3:::my-bucket/*"
      }
    ]
  })
}
