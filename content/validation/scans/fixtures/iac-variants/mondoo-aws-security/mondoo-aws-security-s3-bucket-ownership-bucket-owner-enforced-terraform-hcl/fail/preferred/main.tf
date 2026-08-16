# Non-compliant: ownership set to BucketOwnerPreferred, so ACLs remain enabled.
resource "aws_s3_bucket" "fail_example" {
  bucket = "fail-example-bucket"
}

resource "aws_s3_bucket_ownership_controls" "fail_example" {
  bucket = aws_s3_bucket.fail_example.id

  rule {
    object_ownership = "BucketOwnerPreferred"
  }
}
