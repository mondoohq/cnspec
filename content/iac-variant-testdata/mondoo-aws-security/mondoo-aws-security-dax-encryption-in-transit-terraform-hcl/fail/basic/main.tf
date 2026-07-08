# Non-compliant: DAX cluster does not require TLS for its endpoint.
resource "aws_dax_cluster" "fail_example" {
  cluster_name                     = "fail-example"
  iam_role_arn                     = "arn:aws:iam::123456789012:role/dax"
  node_type                        = "dax.r4.large"
  replication_factor               = 1
  cluster_endpoint_encryption_type = "NONE"
}
