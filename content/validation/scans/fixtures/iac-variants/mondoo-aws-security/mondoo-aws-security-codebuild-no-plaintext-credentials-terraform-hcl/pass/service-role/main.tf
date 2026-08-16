# Compliant: SERVICE_ROLE pull credentials, sensitive value from Secrets Manager.
resource "aws_codebuild_project" "pass_example" {
  name         = "example-project"
  service_role = "arn:aws:iam::111122223333:role/example"

  artifacts {
    type = "NO_ARTIFACTS"
  }

  environment {
    compute_type                = "BUILD_GENERAL1_SMALL"
    image                       = "111122223333.dkr.ecr.us-east-1.amazonaws.com/custom:latest"
    type                        = "LINUX_CONTAINER"
    image_pull_credentials_type = "SERVICE_ROLE"

    environment_variable {
      name  = "DB_PASSWORD"
      value = "arn:aws:secretsmanager:us-east-1:111122223333:secret:db-abc123:password::"
      type  = "SECRETS_MANAGER"
    }
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/example/example.git"
  }
}
