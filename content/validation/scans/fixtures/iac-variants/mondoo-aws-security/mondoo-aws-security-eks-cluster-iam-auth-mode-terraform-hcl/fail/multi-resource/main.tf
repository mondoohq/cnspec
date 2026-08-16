# Non-compliant: two clusters; exactly one uses the deprecated CONFIG_MAP mode.
# .all() over resources must still fail.
resource "aws_eks_cluster" "good" {
  name     = "good"
  role_arn = var.cluster_arn

  access_config {
    authentication_mode = "API"
  }
}

resource "aws_eks_cluster" "bad" {
  name     = "bad"
  role_arn = var.cluster_arn

  access_config {
    authentication_mode = "CONFIG_MAP"
  }
}
