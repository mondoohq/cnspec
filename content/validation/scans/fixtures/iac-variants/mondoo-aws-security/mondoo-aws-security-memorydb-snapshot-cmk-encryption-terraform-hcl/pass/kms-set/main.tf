# Compliant: MemoryDB snapshot encrypted with a customer-managed KMS key.
resource "aws_memorydb_snapshot" "pass_example" {
  name        = "pass-example"
  cluster_name = "example-cluster"
  kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/abcd"
}
