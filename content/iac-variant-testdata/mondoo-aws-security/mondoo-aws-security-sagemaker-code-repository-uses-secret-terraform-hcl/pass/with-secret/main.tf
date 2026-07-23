# Compliant: git_config block references a secret ARN.
resource "aws_sagemaker_code_repository" "pass_example" {
  code_repository_name = "example-repo"

  git_config {
    repository_url = "https://github.com/example/repo.git"
    secret_arn     = "arn:aws:secretsmanager:us-east-1:123456789012:secret:example-abc123"
  }
}
