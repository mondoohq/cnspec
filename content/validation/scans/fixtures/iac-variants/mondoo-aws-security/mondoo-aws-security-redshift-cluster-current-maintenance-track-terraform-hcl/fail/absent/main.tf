# Non-compliant: maintenance_track_name omitted, so the cluster defaults to "current"?
# AWS default is "current"; but with no explicit value the argument is absent and the
# check cannot confirm the track, treating it as non-compliant.
resource "aws_redshift_cluster" "example" {
  cluster_identifier = "example"
  node_type          = "dc2.large"
  master_username    = "admin"
}
