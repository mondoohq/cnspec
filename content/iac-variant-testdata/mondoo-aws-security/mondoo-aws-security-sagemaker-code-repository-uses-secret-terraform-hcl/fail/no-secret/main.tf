# Non-compliant: git_config block has no secret_arn.
resource "aws_sagemaker_code_repository" "fail_example" {
  code_repository_name = "example-repo"

  git_config {
    repository_url = "https://github.com/example/repo.git"
  }
}
