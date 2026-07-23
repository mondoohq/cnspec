# Compliant: build environment does not run in privileged mode.
resource "aws_codebuild_project" "pass_example" {
  name = "example"

  environment {
    compute_type    = "BUILD_GENERAL1_SMALL"
    image           = "aws/codebuild/standard:5.0"
    type            = "LINUX_CONTAINER"
    privileged_mode = false
  }
}
