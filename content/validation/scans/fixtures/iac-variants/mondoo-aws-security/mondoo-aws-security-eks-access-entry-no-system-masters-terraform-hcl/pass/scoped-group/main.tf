# Compliant: the access entry maps the principal to a project group bound to a
# scoped ClusterRole rather than to system:masters.
resource "aws_eks_access_entry" "pass_scoped_group" {
  cluster_name      = "example"
  principal_arn     = "arn:aws:iam::111122223333:role/TeamAViewer"
  type              = "STANDARD"
  kubernetes_groups = ["team-a-viewers"]
}
