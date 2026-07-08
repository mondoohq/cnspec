# Non-compliant: authentication_mode set to CONFIG_MAP only.
resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  access_config {
    authentication_mode = "CONFIG_MAP"
  }
}
