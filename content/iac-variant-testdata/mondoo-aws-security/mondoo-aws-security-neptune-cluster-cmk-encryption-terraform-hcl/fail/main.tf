# Non-compliant: no customer-managed KMS key is set.
resource "aws_neptune_cluster" "fail_example" {
  cluster_identifier = "example"
}
