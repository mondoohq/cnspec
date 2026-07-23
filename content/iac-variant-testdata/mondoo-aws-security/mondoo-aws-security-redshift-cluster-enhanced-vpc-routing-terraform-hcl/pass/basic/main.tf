# Compliant: enhanced VPC routing is enabled.
resource "aws_redshift_cluster" "example" {
  cluster_identifier   = "example"
  node_type            = "dc2.large"
  master_username      = "admin"
  enhanced_vpc_routing = true
}
