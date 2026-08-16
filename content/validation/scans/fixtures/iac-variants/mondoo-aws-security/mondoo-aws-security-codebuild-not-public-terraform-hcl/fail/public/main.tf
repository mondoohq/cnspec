# Non-compliant: project is publicly readable.
resource "aws_codebuild_project" "fail_example" {
  name               = "example"
  project_visibility = "PUBLIC_READ"

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "aws/codebuild/standard:5.0"
    type         = "LINUX_CONTAINER"
  }
}
