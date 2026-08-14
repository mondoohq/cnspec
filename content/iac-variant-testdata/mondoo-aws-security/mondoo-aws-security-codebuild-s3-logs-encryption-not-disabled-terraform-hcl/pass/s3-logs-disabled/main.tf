# S3 logging turned off entirely, so there is no unencrypted log to worry about.
# The check scopes itself to ENABLED s3_logs blocks.
resource "aws_codebuild_project" "app" {
  name         = "app-build"
  service_role = aws_iam_role.codebuild.arn

  artifacts {
    type = "NO_ARTIFACTS"
  }

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "aws/codebuild/amazonlinux2-x86_64-standard:5.0"
    type         = "LINUX_CONTAINER"
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/example/app.git"
  }

  logs_config {
    s3_logs {
      status              = "DISABLED"
      encryption_disabled = true
    }

    cloudwatch_logs {
      status = "ENABLED"
    }
  }
}
