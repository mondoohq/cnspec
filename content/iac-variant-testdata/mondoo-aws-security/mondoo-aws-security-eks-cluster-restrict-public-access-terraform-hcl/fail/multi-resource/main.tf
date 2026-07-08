# Non-compliant: two clusters; exactly one allows 0.0.0.0/0. .all() over
# resources must still fail.
resource "aws_eks_cluster" "good" {
  name     = "good"
  role_arn = var.cluster_arn

  vpc_config {
    public_access_cidrs = ["10.0.0.0/16"]
  }
}

resource "aws_eks_cluster" "bad" {
  name     = "bad"
  role_arn = var.cluster_arn

  vpc_config {
    public_access_cidrs = ["10.0.0.0/16", "0.0.0.0/0"]
  }
}
