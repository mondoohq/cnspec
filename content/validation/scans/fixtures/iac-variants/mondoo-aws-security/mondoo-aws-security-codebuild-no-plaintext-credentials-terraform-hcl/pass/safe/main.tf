# Compliant: CODEBUILD pull credentials and no sensitive plaintext env vars.
resource "aws_codebuild_project" "pass_example" {
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
      name  = "LOG_LEVEL"
      value = "info"
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "DB_PASSWORD"
      value = "/codebuild/db_password"
      type  = "PARAMETER_STORE"
    }
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/example/example.git"
  }
}
