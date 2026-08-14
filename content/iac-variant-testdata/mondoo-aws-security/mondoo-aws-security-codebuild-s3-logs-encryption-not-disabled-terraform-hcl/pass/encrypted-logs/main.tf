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
      status              = "ENABLED"
      location            = "${aws_s3_bucket.build_logs.id}/build-log"
      encryption_disabled = false
    }
  }
}
