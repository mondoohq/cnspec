# Compliant: enabled_cluster_log_types is set.
resource "aws_eks_cluster" "pass_example" {
  name                      = "pass_example_cluster"
  role_arn                  = var.cluster_arn
  enabled_cluster_log_types = ["api", "audit", "authenticator", "controllerManager", "scheduler"]

  vpc_config {
    endpoint_private_access = true
  }
}
