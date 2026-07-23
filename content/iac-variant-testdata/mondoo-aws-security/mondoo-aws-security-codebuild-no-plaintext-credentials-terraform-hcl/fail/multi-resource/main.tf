# Two projects; the second exposes a PLAINTEXT secret, so .all() must fail.
resource "aws_codebuild_project" "safe" {
  name         = "safe-project"
  service_role = "arn:aws:iam::111122223333:role/example"
  artifacts { type = "NO_ARTIFACTS" }
  environment {
    compute_type                = "BUILD_GENERAL1_SMALL"
    image                       = "aws/codebuild/standard:5.0"
    type                        = "LINUX_CONTAINER"
    image_pull_credentials_type = "CODEBUILD"
    environment_variable {
      name  = "DB_PASSWORD"
      value = "arn:aws:secretsmanager:us-east-1:111122223333:secret:db"
      type  = "SECRETS_MANAGER"
    }
  }
  source {
    type     = "GITHUB"
    location = "https://github.com/example/safe.git"
  }
}

resource "aws_codebuild_project" "unsafe" {
  name         = "unsafe-project"
  service_role = "arn:aws:iam::111122223333:role/example"
  artifacts { type = "NO_ARTIFACTS" }
  environment {
    compute_type                = "BUILD_GENERAL1_SMALL"
    image                       = "aws/codebuild/standard:5.0"
    type                        = "LINUX_CONTAINER"
    image_pull_credentials_type = "CODEBUILD"
    environment_variable {
      name  = "DB_PASSWORD"
      value = "hunter2-example"
      type  = "PLAINTEXT"
    }
  }
  source {
    type     = "GITHUB"
    location = "https://github.com/example/unsafe.git"
  }
}
