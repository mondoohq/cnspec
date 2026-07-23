# Non-compliant: deletion_protection omitted, so it defaults to disabled.
resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    endpoint_private_access = true
  }
}
