# Non-compliant: git_config secret_arn is set to an empty string.
resource "aws_sagemaker_code_repository" "fail_example" {
  code_repository_name = "example-repo"

  git_config {
    repository_url = "https://github.com/example/repo.git"
    secret_arn     = ""
  }
}
