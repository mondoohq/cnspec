# Non-compliant: MemoryDB cluster uses the open-access ACL.
resource "aws_memorydb_cluster" "fail_example" {
  name      = "fail-example"
  node_type = "db.t4g.small"
  acl_name  = "open-access"
}
