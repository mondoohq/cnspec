# Non-compliant: enabled_cluster_log_types is not set.
resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    endpoint_private_access = true
  }
}
