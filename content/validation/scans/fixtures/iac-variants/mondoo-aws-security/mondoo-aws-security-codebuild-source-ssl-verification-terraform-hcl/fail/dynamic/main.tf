# Non-compliant: SSL verification disabled via a dynamic source block.
variable "sources" {
  type    = list(string)
  default = ["https://github.com/example/repo.git"]
}

resource "aws_codebuild_project" "fail_dynamic" {
  name = "example"

  environment {
    compute_type = "BUILD_GENERAL1_SMALL"
    image        = "aws/codebuild/standard:5.0"
    type         = "LINUX_CONTAINER"
  }

  dynamic "source" {
    for_each = var.sources
    content {
      type         = "GITHUB"
      location     = source.value
      insecure_ssl = true
    }
  }
}
