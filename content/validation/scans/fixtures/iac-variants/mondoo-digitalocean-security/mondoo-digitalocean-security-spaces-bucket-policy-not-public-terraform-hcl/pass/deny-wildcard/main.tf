resource "digitalocean_spaces_bucket_policy" "example" {
  region = "nyc3"
  bucket = "example-bucket"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Deny"
        Principal = "*"
        Action    = ["s3:GetObject"]
        Resource  = "arn:aws:s3:::example-bucket/*"
      }
    ]
  })
}
