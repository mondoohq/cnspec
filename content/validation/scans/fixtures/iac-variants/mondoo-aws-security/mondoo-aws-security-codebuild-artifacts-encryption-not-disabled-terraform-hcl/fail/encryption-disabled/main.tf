# Build artifacts frequently contain source, dependencies, and occasionally
# credentials; writing them unencrypted removes the last control on the bucket.
resource "aws_codebuild_project" "app" {
  name         = "app-build"
  service_role = aws_iam_role.codebuild.arn

  artifacts {
    type                = "S3"
    location            = aws_s3_bucket.artifacts.bucket
    encryption_disabled = true
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
}
