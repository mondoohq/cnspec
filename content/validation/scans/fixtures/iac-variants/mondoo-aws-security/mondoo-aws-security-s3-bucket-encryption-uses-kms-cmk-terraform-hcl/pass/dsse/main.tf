# Compliant: bucket default encryption uses dual-layer KMS (aws:kms:dsse).
resource "aws_s3_bucket" "pass_dsse" {
  bucket = "pass-dsse-bucket"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "pass_dsse" {
  bucket = aws_s3_bucket.pass_dsse.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms:dsse"
      kms_master_key_id = "arn:aws:kms:us-east-1:111122223333:key/abcd"
    }
  }
}
