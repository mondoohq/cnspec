# Compliant: endpoint access resolved via conditionals whose active branches
# keep the control plane private only.
variable "expose_public" {
  type    = bool
  default = false
}

resource "aws_eks_cluster" "pass_example" {
  name     = "pass_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    endpoint_private_access = var.expose_public ? false : true
    endpoint_public_access  = var.expose_public ? true : false
  }
}
