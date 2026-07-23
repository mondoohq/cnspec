# Compliant: bucket has no static website configuration.
resource "aws_s3_bucket" "example" {
  bucket = "my-private-bucket"
}
