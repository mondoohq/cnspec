# Compliant: cluster is encrypted with a customer-managed KMS key.
resource "aws_neptune_cluster" "pass_example" {
  cluster_identifier = "example"
  kms_key_arn        = "arn:aws:kms:us-east-1:123456789012:key/abcd-1234"
}
