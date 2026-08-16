# Compliant: DocumentDB instance enables auto minor version upgrade.
resource "aws_docdb_cluster_instance" "pass_example" {
  identifier                 = "pass-example"
  cluster_identifier         = "my-cluster"
  instance_class             = "db.r5.large"
  auto_minor_version_upgrade = true
}
