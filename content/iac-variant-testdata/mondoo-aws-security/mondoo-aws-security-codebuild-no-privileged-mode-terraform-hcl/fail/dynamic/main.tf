# Non-compliant: privileged mode set via a dynamic environment block.
variable "envs" {
  type    = list(string)
  default = ["LINUX_CONTAINER"]
}

resource "aws_codebuild_project" "fail_dynamic" {
  name = "example"

  dynamic "environment" {
    for_each = var.envs
    content {
      compute_type    = "BUILD_GENERAL1_SMALL"
      image           = "aws/codebuild/standard:5.0"
      type            = environment.value
      privileged_mode = true
    }
  }
}
