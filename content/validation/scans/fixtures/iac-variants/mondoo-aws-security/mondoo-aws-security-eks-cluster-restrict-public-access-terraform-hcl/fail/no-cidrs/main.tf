# Non-compliant: vpc_config present but public_access_cidrs not set.
resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    endpoint_public_access = true
  }
}
