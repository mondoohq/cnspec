# Compliant: authentication_mode resolved via a conditional whose active branch
# selects API.
variable "use_api" {
  type    = bool
  default = true
}

resource "aws_eks_cluster" "pass_example" {
  name     = "pass_example_cluster"
  role_arn = var.cluster_arn

  access_config {
    authentication_mode = var.use_api ? "API" : "CONFIG_MAP"
  }
}
