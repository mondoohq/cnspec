# Non-compliant: enabled_cluster_log_types explicitly set to an empty list.
resource "aws_eks_cluster" "fail_example" {
  name                      = "fail_example_cluster"
  role_arn                  = var.cluster_arn
  enabled_cluster_log_types = []

  vpc_config {
    endpoint_private_access = true
  }
}
