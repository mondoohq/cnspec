# Non-compliant: Glue data catalog encryption at rest is DISABLED.
resource "aws_glue_data_catalog_encryption_settings" "fail_example" {
  data_catalog_encryption_settings {
    encryption_at_rest {
      catalog_encryption_mode = "DISABLED"
    }
  }
}
