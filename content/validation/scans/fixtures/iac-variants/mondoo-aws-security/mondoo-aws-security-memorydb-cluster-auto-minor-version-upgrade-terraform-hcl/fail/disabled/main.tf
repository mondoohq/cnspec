# Non-compliant: auto minor version upgrade disabled.
resource "aws_memorydb_cluster" "fail_example" {
  name                       = "example"
  node_type                  = "db.t4g.small"
  acl_name                   = "open-access"
  auto_minor_version_upgrade = false
}
