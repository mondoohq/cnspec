# Non-compliant: source disables SSL certificate verification.
resource "aws_codebuild_project" "fail_example" {
  name = "example"

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "aws/codebuild/standard:5.0"
    type         = "LINUX_CONTAINER"
  }

  source {
    type         = "GITHUB"
    location     = "https://github.com/example/repo.git"
    insecure_ssl = true
  }
}
