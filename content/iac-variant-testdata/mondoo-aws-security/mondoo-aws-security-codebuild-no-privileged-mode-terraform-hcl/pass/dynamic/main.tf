# Compliant: dynamic environment block does not enable privileged mode.
variable "envs" {
  type    = list(string)
  default = ["LINUX_CONTAINER"]
}

resource "aws_codebuild_project" "pass_dynamic" {
  name = "example"

  dynamic "environment" {
    for_each = var.envs
    content {
      compute_type    = "BUILD_GENERAL1_SMALL"
      image           = "aws/codebuild/standard:5.0"
      type            = environment.value
      privileged_mode = false
    }
  }
}
