# Non-compliant: two clusters; exactly one omits encryption_config. .all() over
# resources must still fail.
resource "aws_eks_cluster" "good" {
  name     = "good"
  role_arn = var.cluster_arn

  encryption_config {
    resources = ["secrets"]
    provider {
      key_arn = var.kms_arn
    }
  }
}

resource "aws_eks_cluster" "bad" {
  name     = "bad"
  role_arn = var.cluster_arn
}
