# Compliant: source does not disable SSL verification.
resource "aws_codebuild_project" "pass_example" {
  name = "example"

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "aws/codebuild/standard:5.0"
    type         = "LINUX_CONTAINER"
  }

  source {
    type         = "GITHUB"
    location     = "https://github.com/example/repo.git"
    insecure_ssl = false
  }
}
