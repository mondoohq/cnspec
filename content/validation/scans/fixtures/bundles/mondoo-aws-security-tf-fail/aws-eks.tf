# Fail encrypt secrets

resource "aws_eks_cluster" "bad_example" {
  name = "bad_example_cluster"

  role_arn = var.cluster_arn
  vpc_config {
    endpoint_public_access = false
  }
}

# Public Access Check

resource "aws_eks_cluster" "bad_example" {
// other config 
  name = "bad_example_cluster"
  role_arn = var.cluster_arn
  vpc_config {
    endpoint_public_access = true
  }
}