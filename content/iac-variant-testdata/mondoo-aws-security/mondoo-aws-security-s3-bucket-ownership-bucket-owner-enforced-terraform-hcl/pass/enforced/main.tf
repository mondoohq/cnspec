# Compliant: ownership controls enforce BucketOwnerEnforced (ACLs disabled).
resource "aws_s3_bucket" "pass_example" {
  bucket = "pass-example-bucket"
}

resource "aws_s3_bucket_ownership_controls" "pass_example" {
  bucket = aws_s3_bucket.pass_example.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}
