# Non-compliant: authentication_mode resolved via a conditional whose active
# branch selects the deprecated CONFIG_MAP mode.
variable "use_api" {
  type    = bool
  default = false
}

resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  access_config {
    authentication_mode = var.use_api ? "API" : "CONFIG_MAP"
  }
}
