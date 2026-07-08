# Compliant: bucket has a server-side encryption configuration with a default rule.
resource "aws_s3_bucket" "pass_example" {
  bucket = "pass-example-bucket"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "pass_example" {
  bucket = aws_s3_bucket.pass_example.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}
