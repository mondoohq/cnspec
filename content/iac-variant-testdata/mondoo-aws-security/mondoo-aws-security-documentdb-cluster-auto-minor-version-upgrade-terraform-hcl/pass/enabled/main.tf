# Compliant: DocumentDB cluster instance enables automatic minor version upgrades.
resource "aws_docdb_cluster_instance" "pass_example" {
  identifier                 = "example-instance"
  cluster_identifier         = "example-cluster"
  instance_class             = "db.r5.large"
  auto_minor_version_upgrade = true
}
