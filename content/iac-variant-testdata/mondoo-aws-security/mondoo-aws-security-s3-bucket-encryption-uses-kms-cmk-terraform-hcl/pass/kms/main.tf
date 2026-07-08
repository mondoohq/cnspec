# Compliant: bucket default encryption uses a KMS key.
resource "aws_s3_bucket" "pass_example" {
  bucket = "pass-example-bucket"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "pass_example" {
  bucket = aws_s3_bucket.pass_example.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = "arn:aws:kms:us-east-1:111122223333:key/abcd"
    }
  }
}
