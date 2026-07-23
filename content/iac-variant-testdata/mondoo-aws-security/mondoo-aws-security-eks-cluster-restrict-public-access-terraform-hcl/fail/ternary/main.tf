# Non-compliant: public_access_cidrs resolved via a conditional whose active
# branch opens the endpoint to the world.
variable "open_to_world" {
  type    = bool
  default = true
}

resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  vpc_config {
    public_access_cidrs = var.open_to_world ? ["0.0.0.0/0"] : ["10.0.0.0/16"]
  }
}
