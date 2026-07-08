# Compliant: MSK cluster encrypts data at rest with a customer-managed KMS key.
resource "aws_msk_cluster" "pass_example" {
  cluster_name = "pass-example"

  encryption_info {
    encryption_at_rest_kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/abcd"
  }
}
