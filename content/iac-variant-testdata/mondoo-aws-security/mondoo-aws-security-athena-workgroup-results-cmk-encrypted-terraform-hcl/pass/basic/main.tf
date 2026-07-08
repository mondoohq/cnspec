# Compliant: query results encrypted with a customer-managed KMS key.
resource "aws_athena_workgroup" "pass_example" {
  name = "pass-example"

  configuration {
    result_configuration {
      encryption_configuration {
        encryption_option = "SSE_KMS"
        kms_key_arn       = "arn:aws:kms:us-east-1:111122223333:key/abcd"
      }
    }
  }
}
