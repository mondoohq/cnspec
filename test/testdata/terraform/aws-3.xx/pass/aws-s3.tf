resource "aws_s3_bucket" "pass_example" {
  bucket = "test_bucket"
  acl = "private"

  versioning {
    enabled = true
  }
  logging {
    target_bucket = "bucket-name"
  }

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.mykey.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }
}

resource "aws_s3_bucket_public_access_block" "pass_example" {
  bucket = aws_s3_bucket.pass_example.id

  block_public_policy = true
  block_public_acls = true
  ignore_public_acls = true
  restrict_public_buckets = true
}
