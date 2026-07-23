# Non-compliant: deletion protection is disabled.
resource "aws_eks_cluster" "fail_example" {
  name                = "fail_example_cluster"
  role_arn            = var.cluster_arn
  deletion_protection = false

  vpc_config {
    endpoint_private_access = true
  }
}
