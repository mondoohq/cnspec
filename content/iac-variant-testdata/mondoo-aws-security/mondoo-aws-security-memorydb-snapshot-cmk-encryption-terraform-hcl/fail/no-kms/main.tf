# Non-compliant: MemoryDB snapshot has no customer-managed KMS key.
resource "aws_memorydb_snapshot" "fail_example" {
  name        = "fail-example"
  cluster_name = "example-cluster"
}
