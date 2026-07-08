# Non-compliant: cluster is publicly accessible.
resource "aws_redshift_cluster" "example" {
  cluster_identifier  = "example"
  node_type           = "dc2.large"
  master_username     = "admin"
  publicly_accessible = true
}
