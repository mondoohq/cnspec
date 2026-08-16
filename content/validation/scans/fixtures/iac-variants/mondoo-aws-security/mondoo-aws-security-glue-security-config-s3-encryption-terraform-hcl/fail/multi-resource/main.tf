# Non-compliant: two security configurations, one with S3 encryption DISABLED.
# .all() must still fail when any single resource violates.
resource "aws_glue_security_configuration" "good" {
  name = "good-config"

  encryption_configuration {
    s3_encryption {
      s3_encryption_mode = "SSE-KMS"
      kms_key_arn        = "arn:aws:kms:us-east-1:123456789012:key/abc"
    }
  }
}

resource "aws_glue_security_configuration" "bad" {
  name = "bad-config"

  encryption_configuration {
    s3_encryption {
      s3_encryption_mode = "DISABLED"
    }
  }
}
