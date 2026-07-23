# Non-compliant: DocumentDB cluster instance disables automatic minor version upgrades.
resource "aws_docdb_cluster_instance" "fail_example" {
  identifier                 = "example-instance"
  cluster_identifier         = "example-cluster"
  instance_class             = "db.r5.large"
  auto_minor_version_upgrade = false
}
