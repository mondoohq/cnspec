# Compliant: the access entry declares no Kubernetes groups, so authorization comes
# from a scoped EKS access policy association instead of a group binding.
resource "aws_eks_access_entry" "pass_no_groups" {
  cluster_name  = "example"
  principal_arn = "arn:aws:iam::111122223333:role/ClusterAdmin"
  type          = "STANDARD"
}
