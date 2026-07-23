# Non-compliant: bucket default encryption uses AES256, not a KMS CMK.
resource "aws_s3_bucket" "fail_example" {
  bucket = "fail-example-bucket"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "fail_example" {
  bucket = aws_s3_bucket.fail_example.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}
