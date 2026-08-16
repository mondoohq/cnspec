# Compliant: DAX cluster has server-side encryption enabled.
resource "aws_dax_cluster" "pass_example" {
  cluster_name       = "pass-example"
  iam_role_arn       = "arn:aws:iam::123456789012:role/dax"
  node_type          = "dax.r4.large"
  replication_factor = 1

  server_side_encryption {
    enabled = true
  }
}
