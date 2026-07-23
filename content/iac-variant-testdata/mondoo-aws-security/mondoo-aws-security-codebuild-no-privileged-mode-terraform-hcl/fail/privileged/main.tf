# Non-compliant: build environment runs in privileged mode.
resource "aws_codebuild_project" "fail_example" {
  name = "example"

  environment {
    compute_type    = "BUILD_GENERAL1_SMALL"
    image           = "aws/codebuild/standard:5.0"
    type            = "LINUX_CONTAINER"
    privileged_mode = true
  }
}
