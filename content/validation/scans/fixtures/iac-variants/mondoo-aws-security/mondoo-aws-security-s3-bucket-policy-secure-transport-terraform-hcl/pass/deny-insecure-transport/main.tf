resource "aws_s3_bucket_policy" "example" {
  bucket = "my-bucket"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource  = "arn:aws:s3:::my-bucket/*"
        Condition = { Bool = { "aws:SecureTransport" = "false" } }
      }
    ]
  })
}
