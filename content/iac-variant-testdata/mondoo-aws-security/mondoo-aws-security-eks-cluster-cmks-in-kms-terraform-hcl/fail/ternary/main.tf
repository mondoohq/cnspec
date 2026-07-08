# Non-compliant: key_arn resolved through a conditional whose active branch is
# an empty string.
variable "use_cmk" {
  type    = bool
  default = false
}

resource "aws_eks_cluster" "fail_example" {
  name     = "fail_example_cluster"
  role_arn = var.cluster_arn

  encryption_config {
    resources = ["secrets"]
    provider {
      key_arn = var.use_cmk ? var.kms_arn : ""
    }
  }
}
