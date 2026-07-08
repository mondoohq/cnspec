# Non-compliant: encryption not enabled (encrypted omitted; historically defaults off).
resource "aws_redshift_cluster" "example" {
  cluster_identifier = "example"
  node_type          = "dc2.large"
  master_username    = "admin"
}
