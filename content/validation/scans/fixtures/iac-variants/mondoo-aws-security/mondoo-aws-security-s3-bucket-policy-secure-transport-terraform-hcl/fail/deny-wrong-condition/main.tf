resource "aws_s3_bucket_policy" "example" {
  bucket = "my-bucket"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "DenyOutsideVpc"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource  = "arn:aws:s3:::my-bucket/*"
        Condition = { StringNotEquals = { "aws:SourceVpc" = "vpc-111222333" } }
      }
    ]
  })
}
