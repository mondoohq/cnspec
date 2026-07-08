# Non-compliant: image pull credentials type is not CODEBUILD or SERVICE_ROLE.
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
    image_pull_credentials_type = "CUSTOM"

    environment_variable {
      name  = "LOG_LEVEL"
      value = "info"
      type  = "PLAINTEXT"
    }
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/example/example.git"
  }
}
