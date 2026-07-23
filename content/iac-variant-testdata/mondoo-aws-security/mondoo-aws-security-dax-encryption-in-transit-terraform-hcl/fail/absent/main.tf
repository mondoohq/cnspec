# Non-compliant: cluster_endpoint_encryption_type is omitted, so it defaults to
# NONE and TLS is not required.
resource "aws_dax_cluster" "fail_absent" {
  cluster_name       = "fail-absent"
  iam_role_arn       = "arn:aws:iam::123456789012:role/dax"
  node_type          = "dax.r4.large"
  replication_factor = 1
}
