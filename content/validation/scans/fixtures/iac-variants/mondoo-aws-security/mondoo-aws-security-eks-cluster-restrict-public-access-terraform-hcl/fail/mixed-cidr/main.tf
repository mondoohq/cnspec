# Non-compliant: public_access_cidrs includes 0.0.0.0/0 alongside a real range.
resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    endpoint_public_access = true
    public_access_cidrs    = ["10.0.0.0/16", "0.0.0.0/0"]
  }
}
