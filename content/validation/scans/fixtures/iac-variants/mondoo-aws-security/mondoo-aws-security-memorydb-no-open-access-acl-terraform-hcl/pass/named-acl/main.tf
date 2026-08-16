# Compliant: MemoryDB cluster uses a scoped ACL.
resource "aws_memorydb_cluster" "pass_example" {
  name      = "pass-example"
  node_type = "db.t4g.small"
  acl_name  = "my-restricted-acl"
}
