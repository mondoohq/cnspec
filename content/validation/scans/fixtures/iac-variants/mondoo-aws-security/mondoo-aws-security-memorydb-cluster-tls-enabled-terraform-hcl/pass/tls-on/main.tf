# Compliant: MemoryDB cluster has TLS enabled.
resource "aws_memorydb_cluster" "pass_example" {
  name        = "pass-example"
  node_type   = "db.t4g.small"
  acl_name    = "my-acl"
  tls_enabled = true
}
