# Non-compliant: MemoryDB cluster has TLS disabled.
resource "aws_memorydb_cluster" "fail_example" {
  name        = "fail-example"
  node_type   = "db.t4g.small"
  acl_name    = "my-acl"
  tls_enabled = false
}
