resource "aws_athena_workgroup" "analytics" {
  name = "analytics"

  configuration {
    enforce_workgroup_configuration    = true
    enable_minimum_encryption_configuration = true

    result_configuration {
      output_location = "s3://example-athena-results/analytics/"

      encryption_configuration {
        encryption_option = "SSE_KMS"
        kms_key_arn       = aws_kms_key.athena.arn
      }
    }
  }
}
