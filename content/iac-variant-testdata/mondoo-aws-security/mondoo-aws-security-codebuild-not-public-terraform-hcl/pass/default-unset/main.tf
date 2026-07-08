# Compliant: project_visibility is not set and defaults to PRIVATE.
resource "aws_codebuild_project" "pass_default" {
  name         = "example"
  service_role = aws_iam_role.example.arn

  artifacts {
    type = "NO_ARTIFACTS"
  }

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "aws/codebuild/standard:5.0"
    type         = "LINUX_CONTAINER"
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/example/repo.git"
  }
}
