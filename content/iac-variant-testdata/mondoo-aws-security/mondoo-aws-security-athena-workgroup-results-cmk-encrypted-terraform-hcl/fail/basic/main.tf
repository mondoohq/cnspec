# Non-compliant: query results use SSE_S3 instead of a KMS key.
resource "aws_athena_workgroup" "fail_example" {
  name = "fail-example"

  configuration {
    result_configuration {
      encryption_configuration {
        encryption_option = "SSE_S3"
      }
    }
  }
}
