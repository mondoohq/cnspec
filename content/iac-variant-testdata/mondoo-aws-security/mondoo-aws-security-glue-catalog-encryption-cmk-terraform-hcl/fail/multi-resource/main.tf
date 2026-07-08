# Two catalog encryption settings; the second lacks a CMK, so .all() must fail.
resource "aws_glue_data_catalog_encryption_settings" "compliant" {
  data_catalog_encryption_settings {
    encryption_at_rest {
      catalog_encryption_mode = "SSE-KMS"
      sse_aws_kms_key_id      = "arn:aws:kms:us-east-1:111122223333:key/abcd-1234"
    }
  }
}

resource "aws_glue_data_catalog_encryption_settings" "violating" {
  data_catalog_encryption_settings {
    encryption_at_rest {
      catalog_encryption_mode = "SSE-KMS"
    }
  }
}
