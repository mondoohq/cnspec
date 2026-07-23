# Non-compliant: one of two projects runs in privileged mode.
resource "aws_codebuild_project" "ok" {
  name = "ok"
  environment {
    compute_type    = "BUILD_GENERAL1_SMALL"
    image           = "aws/codebuild/standard:5.0"
    type            = "LINUX_CONTAINER"
    privileged_mode = false
  }
}

resource "aws_codebuild_project" "bad" {
  name = "bad"
  environment {
    compute_type    = "BUILD_GENERAL1_SMALL"
    image           = "aws/codebuild/standard:5.0"
    type            = "LINUX_CONTAINER"
    privileged_mode = true
  }
}
