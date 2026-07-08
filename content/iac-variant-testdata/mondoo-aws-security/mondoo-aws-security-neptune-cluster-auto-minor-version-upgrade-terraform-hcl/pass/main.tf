# Compliant: auto minor version upgrades are enabled.
resource "aws_neptune_cluster_instance" "pass_example" {
  cluster_identifier         = "example"
  instance_class             = "db.r5.large"
  auto_minor_version_upgrade = true
}
