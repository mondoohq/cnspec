# Non-compliant: result configuration has no encryption_configuration block.
resource "aws_athena_workgroup" "fail_no_encryption" {
  name = "fail-no-encryption"

  configuration {
    result_configuration {
      output_location = "s3://example-bucket/output/"
    }
  }
}
