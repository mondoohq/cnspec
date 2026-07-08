# Non-compliant: DAX cluster server-side encryption is disabled.
resource "aws_dax_cluster" "fail_example" {
  cluster_name       = "fail-example"
  iam_role_arn       = "arn:aws:iam::123456789012:role/dax"
  node_type          = "dax.r4.large"
  replication_factor = 1

  server_side_encryption {
    enabled = false
  }
}
