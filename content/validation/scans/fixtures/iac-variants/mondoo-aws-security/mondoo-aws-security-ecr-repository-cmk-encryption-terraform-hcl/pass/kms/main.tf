# Compliant: repository encrypted with a customer-managed KMS key.
resource "aws_ecr_repository" "pass_example" {
  name = "pass-example"

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = "arn:aws:kms:us-east-1:111122223333:key/abcd"
  }
}
