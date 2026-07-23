# Non-compliant: auto minor version upgrades are disabled.
resource "aws_neptune_cluster_instance" "fail_example" {
  cluster_identifier         = "example"
  instance_class             = "db.r5.large"
  auto_minor_version_upgrade = false
}
