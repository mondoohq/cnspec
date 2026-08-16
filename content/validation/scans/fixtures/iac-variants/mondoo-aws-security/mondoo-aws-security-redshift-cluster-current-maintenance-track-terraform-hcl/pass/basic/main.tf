# Compliant: cluster tracks the Current maintenance track.
resource "aws_redshift_cluster" "example" {
  cluster_identifier     = "example"
  node_type              = "dc2.large"
  master_username        = "admin"
  maintenance_track_name = "Current"
}
