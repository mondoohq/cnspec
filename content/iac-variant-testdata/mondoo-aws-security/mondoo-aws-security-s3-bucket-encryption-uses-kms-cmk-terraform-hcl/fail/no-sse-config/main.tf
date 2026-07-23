# Non-compliant: bucket has no server-side encryption configuration resource,
# so it is not encrypted with a KMS CMK.
resource "aws_s3_bucket" "fail_no_sse" {
  bucket = "fail-no-sse-bucket"
}
