resource "aws_s3_bucket_policy" "example" {
  bucket = "my-bucket"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { Service = "cloudfront.amazonaws.com" }
        Action    = "s3:GetObject"
        Resource  = "arn:aws:s3:::my-bucket/*"
      }
    ]
  })
}
