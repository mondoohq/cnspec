# Compliant: control plane endpoint is private only.
resource "aws_eks_cluster" "pass_example" {
  name     = "pass_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    endpoint_private_access = true
    endpoint_public_access  = false
  }
}
