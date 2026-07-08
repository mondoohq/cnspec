# Non-compliant: a counted DAX cluster disables encryption at rest.
resource "aws_dax_cluster" "counted" {
  count              = 2
  cluster_name       = "example-${count.index}"
  iam_role_arn       = "arn:aws:iam::123456789012:role/dax"
  node_type          = "dax.r4.large"
  replication_factor = 1
  server_side_encryption {
    enabled = false
  }
}
