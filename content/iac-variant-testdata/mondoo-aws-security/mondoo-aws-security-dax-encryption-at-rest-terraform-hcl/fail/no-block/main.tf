# Non-compliant: DAX cluster has no server_side_encryption block, so encryption
# at rest is not enabled.
resource "aws_dax_cluster" "fail_no_block" {
  cluster_name       = "fail-no-block"
  iam_role_arn       = "arn:aws:iam::123456789012:role/dax"
  node_type          = "dax.r4.large"
  replication_factor = 1
}
