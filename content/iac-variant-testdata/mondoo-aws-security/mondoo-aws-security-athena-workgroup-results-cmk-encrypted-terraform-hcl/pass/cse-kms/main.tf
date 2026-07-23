# Compliant: query results encrypted with client-side encryption using a KMS key.
resource "aws_athena_workgroup" "pass_cse" {
  name = "pass-cse"

  configuration {
    result_configuration {
      encryption_configuration {
        encryption_option = "CSE_KMS"
        kms_key_arn       = "arn:aws:kms:us-east-1:111122223333:key/abcd"
      }
    }
  }
}
