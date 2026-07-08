# Compliant: public_access_cidrs restricted to a specific range.
resource "aws_eks_cluster" "pass_example" {
  name     = "pass_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    public_access_cidrs = ["10.0.0.0/16", "192.168.1.0/24"]
  }
}
