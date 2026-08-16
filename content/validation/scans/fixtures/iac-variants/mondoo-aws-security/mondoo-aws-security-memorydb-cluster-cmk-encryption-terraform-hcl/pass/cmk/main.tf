# Compliant: cluster encrypted with a customer-managed KMS key.
resource "aws_memorydb_cluster" "pass_example" {
  name        = "example"
  node_type   = "db.t4g.small"
  acl_name    = "open-access"
  kms_key_arn = "arn:aws:kms:us-east-1:123456789012:key/abcd-1234"
}
