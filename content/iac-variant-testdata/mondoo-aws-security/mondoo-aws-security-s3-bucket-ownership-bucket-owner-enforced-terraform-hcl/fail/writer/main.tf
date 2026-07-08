# Non-compliant: ownership set to BucketOwnerObjectWriter, so ACLs remain enabled.
resource "aws_s3_bucket" "fail_writer" {
  bucket = "fail-writer-bucket"
}

resource "aws_s3_bucket_ownership_controls" "fail_writer" {
  bucket = aws_s3_bucket.fail_writer.id

  rule {
    object_ownership = "BucketOwnerObjectWriter"
  }
}
