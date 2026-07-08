# Non-compliant: managed master password is enabled but no customer-managed KMS key is set.
resource "aws_redshift_cluster" "fail_example" {
  cluster_identifier     = "example-cluster"
  node_type              = "ra3.xlplus"
  master_username        = "admin"
  manage_master_password = true
}
