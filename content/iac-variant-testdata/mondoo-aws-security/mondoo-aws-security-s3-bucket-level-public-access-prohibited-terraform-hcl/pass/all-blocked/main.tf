# Compliant: all four public access block settings are enabled.
resource "aws_s3_bucket_public_access_block" "pass_example" {
  bucket                  = "pass-example-bucket"
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
