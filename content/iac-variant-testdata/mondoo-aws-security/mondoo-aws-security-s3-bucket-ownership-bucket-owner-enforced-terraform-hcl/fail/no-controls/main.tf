# Non-compliant: bucket has no ownership controls resource, so BucketOwnerEnforced
# is not applied and ACLs remain enabled.
resource "aws_s3_bucket" "fail_no_controls" {
  bucket = "fail-no-controls-bucket"
}
