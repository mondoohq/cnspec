# Compliant: authentication_mode set to API_AND_CONFIG_MAP.
resource "aws_eks_cluster" "pass_example" {
  name     = "pass_example_cluster"
  role_arn = var.cluster_arn

  access_config {
    authentication_mode = "API_AND_CONFIG_MAP"
  }
}
