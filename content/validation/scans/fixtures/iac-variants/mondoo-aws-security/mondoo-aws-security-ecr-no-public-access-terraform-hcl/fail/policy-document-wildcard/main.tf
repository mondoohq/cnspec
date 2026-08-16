data "aws_iam_policy_document" "ecr" {
  statement {
    effect  = "Allow"
    actions = ["ecr:BatchGetImage"]

    principals {
      type        = "*"
      identifiers = ["*"]
    }
  }
}

resource "aws_ecr_repository" "this" {
  name = "example-repo"
}

resource "aws_ecr_repository_policy" "fail" {
  repository = aws_ecr_repository.this.name
  policy     = data.aws_iam_policy_document.ecr.json
}
