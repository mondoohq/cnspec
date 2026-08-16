# Non-compliant: several plaintext env vars; the last one exposes a secret, so
# the .none() must still fail (catches any first-var-only bug).
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
      name  = "AWS_REGION"
      value = "us-east-1"
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "LOG_LEVEL"
      value = "info"
      type  = "PLAINTEXT"
    }

    environment_variable {
      name  = "DB_PASSWORD"
      value = "hunter2-example"
      type  = "PLAINTEXT"
    }
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/example/example.git"
  }
}
