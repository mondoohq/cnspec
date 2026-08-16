# Non-compliant: DocumentDB instance disables auto minor version upgrade.
resource "aws_docdb_cluster_instance" "fail_example" {
  identifier                 = "fail-example"
  cluster_identifier         = "my-cluster"
  instance_class             = "db.r5.large"
  auto_minor_version_upgrade = false
}
