# Non-compliant: SSE-KMS is enabled but no customer-managed key is set,
# so encryption falls back to the AWS-managed aws/glue key instead of a CMK.
resource "aws_glue_data_catalog_encryption_settings" "fail_example" {
  data_catalog_encryption_settings {
    encryption_at_rest {
      catalog_encryption_mode = "SSE-KMS"
    }
  }
}
