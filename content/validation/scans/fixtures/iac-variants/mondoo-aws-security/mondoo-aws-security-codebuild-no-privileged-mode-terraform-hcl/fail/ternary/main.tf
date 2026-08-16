# Non-compliant: privileged mode enabled through a conditional expression.
variable "privileged" {
  type    = bool
  default = true
}

resource "aws_codebuild_project" "ternary" {
  name = "example"
  environment {
    compute_type    = "BUILD_GENERAL1_SMALL"
    image           = "aws/codebuild/standard:5.0"
    type            = "LINUX_CONTAINER"
    privileged_mode = var.privileged ? true : false
  }
}
