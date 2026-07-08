# Compliant: deletion protection is enabled.
resource "aws_eks_cluster" "pass_example" {
  name                = "pass_example_cluster"
  role_arn            = var.cluster_arn
  deletion_protection = true

  vpc_config {
    endpoint_private_access = true
  }
}
