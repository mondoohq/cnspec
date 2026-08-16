# Non-compliant: public_access_cidrs allows 0.0.0.0/0.
resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    public_access_cidrs = ["0.0.0.0/0"]
  }
}
