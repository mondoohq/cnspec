# Compliant: cluster does not use Secrets Manager managed password, so the CMK requirement does not apply.
resource "aws_redshift_cluster" "pass_example" {
  cluster_identifier     = "example-cluster"
  node_type              = "ra3.xlplus"
  master_username        = "admin"
  manage_master_password = false
}
