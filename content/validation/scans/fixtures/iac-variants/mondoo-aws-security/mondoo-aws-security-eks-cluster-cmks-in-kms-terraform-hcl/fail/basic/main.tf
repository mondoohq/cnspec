# Non-compliant: EKS cluster has no secrets encryption configuration.
resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    endpoint_public_access  = true
    endpoint_private_access = false
  }
}
