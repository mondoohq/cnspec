# Non-compliant: restrict_public_buckets is disabled.
resource "aws_s3_bucket_public_access_block" "fail_example" {
  bucket                  = "fail-example-bucket"
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = false
}
