# Non-compliant: two clusters; exactly one exposes a public endpoint. .all()
# over resources must still fail.
resource "aws_eks_cluster" "good" {
  name     = "good"
  role_arn = var.cluster_arn

  vpc_config {
    endpoint_private_access = true
    endpoint_public_access  = false
  }
}

resource "aws_eks_cluster" "bad" {
  name     = "bad"
  role_arn = var.cluster_arn

  vpc_config {
    endpoint_private_access = true
    endpoint_public_access  = true
  }
}
