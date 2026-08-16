# Non-compliant: no KMS key is set.
resource "aws_redshift_cluster" "example" {
  cluster_identifier = "example"
  node_type          = "dc2.large"
  master_username    = "admin"
}
