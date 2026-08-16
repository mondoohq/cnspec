# Non-compliant: an API token stored as a PLAINTEXT environment variable.
resource "aws_codebuild_project" "fail_example" {
  name         = "example-project"
  service_role = "arn:aws:iam::111122223333:role/example"

  artifacts {
    type = "NO_ARTIFACTS"
  }

  environment {
    compute_type                = "BUILD_GENERAL1_SMALL"
    image                       = "aws/codebuild/standard:5.0"
    type                        = "LINUX_CONTAINER"
    image_pull_credentials_type = "CODEBUILD"

    environment_variable {
      name  = "GITHUB_API_TOKEN"
      value = "ghp_exampletoken0123456789"
      type  = "PLAINTEXT"
    }
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/example/example.git"
  }
}
