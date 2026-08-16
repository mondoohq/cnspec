# Non-compliant: one of two DAX clusters disables encryption at rest.
resource "aws_dax_cluster" "ok" {
  cluster_name       = "ok"
  iam_role_arn       = "arn:aws:iam::123456789012:role/dax"
  node_type          = "dax.r4.large"
  replication_factor = 1
  server_side_encryption {
    enabled = true
  }
}

resource "aws_dax_cluster" "bad" {
  cluster_name       = "bad"
  iam_role_arn       = "arn:aws:iam::123456789012:role/dax"
  node_type          = "dax.r4.large"
  replication_factor = 1
  server_side_encryption {
    enabled = false
  }
}
