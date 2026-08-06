# Non-compliant: the access entry grants unbounded cluster-admin via system:masters.
resource "aws_eks_access_entry" "fail_system_masters" {
  cluster_name      = "example"
  principal_arn     = "arn:aws:iam::111122223333:role/ClusterAdmin"
  type              = "STANDARD"
  kubernetes_groups = ["system:masters"]
}
