# Compliant: project visibility is private.
resource "aws_codebuild_project" "pass_example" {
  name               = "example"
  project_visibility = "PRIVATE"

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "aws/codebuild/standard:5.0"
    type         = "LINUX_CONTAINER"
  }
}
