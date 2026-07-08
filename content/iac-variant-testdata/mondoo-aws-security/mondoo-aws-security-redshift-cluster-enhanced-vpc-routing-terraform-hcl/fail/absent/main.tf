# Non-compliant: enhanced_vpc_routing omitted, so it defaults to disabled.
resource "aws_redshift_cluster" "example" {
  cluster_identifier = "example"
  node_type          = "dc2.large"
  master_username    = "admin"
}
