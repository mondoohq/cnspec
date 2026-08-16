# Non-compliant: block_public_acls is omitted (defaults to false), so public
# ACLs are not fully blocked.
resource "aws_s3_bucket_public_access_block" "fail_absent" {
  bucket                  = "fail-absent-bucket"
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
