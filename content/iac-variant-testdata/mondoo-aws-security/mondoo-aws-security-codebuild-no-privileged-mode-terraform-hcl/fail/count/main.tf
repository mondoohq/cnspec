# Non-compliant: a counted project runs in privileged mode.
resource "aws_codebuild_project" "counted" {
  count = 2
  name  = "example-${count.index}"
  environment {
    compute_type    = "BUILD_GENERAL1_SMALL"
    image           = "aws/codebuild/standard:5.0"
    type            = "LINUX_CONTAINER"
    privileged_mode = true
  }
}
