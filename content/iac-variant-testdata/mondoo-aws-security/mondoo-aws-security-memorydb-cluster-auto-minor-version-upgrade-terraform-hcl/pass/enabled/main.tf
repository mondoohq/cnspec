# Compliant: auto minor version upgrade enabled.
resource "aws_memorydb_cluster" "pass_example" {
  name                     = "example"
  node_type                = "db.t4g.small"
  acl_name                 = "open-access"
  auto_minor_version_upgrade = true
}
