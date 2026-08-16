# Compliant: CodeBuild project encrypts artifacts with a referenced KMS key.
resource "aws_kms_key" "codebuild" {
  description             = "CodeBuild artifact encryption key"
  deletion_window_in_days = 10
}

resource "aws_codebuild_project" "pass_example" {
  name           = "example-project"
  service_role   = "arn:aws:iam::111122223333:role/example"
  encryption_key = aws_kms_key.codebuild.arn

  artifacts {
    type = "NO_ARTIFACTS"
  }

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "aws/codebuild/standard:5.0"
    type         = "LINUX_CONTAINER"
  }

  source {
    type     = "GITHUB"
    location = "https://github.com/example/example.git"
  }
}
