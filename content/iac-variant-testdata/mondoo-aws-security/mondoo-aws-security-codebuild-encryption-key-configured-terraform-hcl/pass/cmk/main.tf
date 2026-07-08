# Compliant: CodeBuild project encrypts artifacts with a KMS key.
resource "aws_codebuild_project" "pass_example" {
  name           = "example-project"
  service_role   = "arn:aws:iam::111122223333:role/example"
  encryption_key = "arn:aws:kms:us-east-1:111122223333:key/abcd1234-a123-456a-a12b-a123b4cd56ef"

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
