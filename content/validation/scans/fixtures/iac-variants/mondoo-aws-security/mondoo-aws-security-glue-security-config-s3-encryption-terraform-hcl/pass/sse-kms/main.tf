resource "aws_glue_security_configuration" "example" {
  name = "example-security-config"

  encryption_configuration {
    s3_encryption {
      s3_encryption_mode = "SSE-KMS"
      kms_key_arn        = "arn:aws:kms:us-east-1:123456789012:key/abc"
    }
  }
}
