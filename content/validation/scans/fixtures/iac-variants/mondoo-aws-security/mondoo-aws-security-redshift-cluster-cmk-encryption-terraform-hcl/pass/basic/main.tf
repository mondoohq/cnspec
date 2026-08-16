# Compliant: cluster uses a customer-managed KMS key.
resource "aws_redshift_cluster" "example" {
  cluster_identifier = "example"
  node_type          = "dc2.large"
  master_username    = "admin"
  kms_key_id         = "arn:aws:kms:us-east-1:123456789012:key/abcd-1234"
}
