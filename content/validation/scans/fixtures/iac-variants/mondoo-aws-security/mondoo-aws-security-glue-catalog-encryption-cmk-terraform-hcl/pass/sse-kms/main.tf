# Compliant: Glue data catalog uses SSE-KMS encryption at rest with a CMK.
resource "aws_glue_data_catalog_encryption_settings" "pass_example" {
  data_catalog_encryption_settings {
    encryption_at_rest {
      catalog_encryption_mode = "SSE-KMS"
      sse_aws_kms_key_id      = "arn:aws:kms:us-east-1:111122223333:key/abcd-1234"
    }
  }
}
