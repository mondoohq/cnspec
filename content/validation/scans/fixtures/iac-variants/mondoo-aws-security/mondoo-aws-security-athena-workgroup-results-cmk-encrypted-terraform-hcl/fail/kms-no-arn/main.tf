# Non-compliant: SSE_KMS selected but no KMS key ARN is supplied.
resource "aws_athena_workgroup" "fail_kms_no_arn" {
  name = "fail-kms-no-arn"

  configuration {
    result_configuration {
      encryption_configuration {
        encryption_option = "SSE_KMS"
      }
    }
  }
}
