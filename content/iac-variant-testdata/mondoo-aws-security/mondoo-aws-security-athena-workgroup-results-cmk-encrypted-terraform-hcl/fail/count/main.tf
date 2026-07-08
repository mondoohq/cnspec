# Non-compliant: a counted workgroup encrypts results with SSE_S3, not a KMS key.
resource "aws_athena_workgroup" "fail_count" {
  count = 2
  name  = "fail-example-${count.index}"

  configuration {
    result_configuration {
      encryption_configuration {
        encryption_option = "SSE_S3"
      }
    }
  }
}
