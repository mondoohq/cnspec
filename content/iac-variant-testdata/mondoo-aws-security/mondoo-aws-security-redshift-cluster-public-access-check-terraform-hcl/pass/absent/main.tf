# Compliant: publicly_accessible omitted, so the cluster defaults to private.
resource "aws_redshift_cluster" "example" {
  cluster_identifier = "example"
  node_type          = "dc2.large"
  master_username    = "admin"
}
