resource "aws_glue_security_configuration" "example" {
  name = "example-security-config"

  encryption_configuration {
    s3_encryption {
      s3_encryption_mode = "SSE-S3"
    }
  }
}
