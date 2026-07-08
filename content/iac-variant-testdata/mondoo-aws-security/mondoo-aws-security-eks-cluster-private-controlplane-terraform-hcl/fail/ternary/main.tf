# Non-compliant: public access enabled via a conditional whose active branch is
# true.
variable "expose_public" {
  type    = bool
  default = true
}

resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    endpoint_private_access = true
    endpoint_public_access  = var.expose_public ? true : false
  }
}
