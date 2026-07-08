# Non-compliant: one of two projects disables SSL verification.
resource "aws_codebuild_project" "ok" {
  name = "ok"
  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "aws/codebuild/standard:5.0"
    type         = "LINUX_CONTAINER"
  }
  source {
    type         = "GITHUB"
    location     = "https://github.com/example/ok.git"
    insecure_ssl = false
  }
}

resource "aws_codebuild_project" "bad" {
  name = "bad"
  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "aws/codebuild/standard:5.0"
    type         = "LINUX_CONTAINER"
  }
  source {
    type         = "GITHUB"
    location     = "https://github.com/example/bad.git"
    insecure_ssl = true
  }
}
