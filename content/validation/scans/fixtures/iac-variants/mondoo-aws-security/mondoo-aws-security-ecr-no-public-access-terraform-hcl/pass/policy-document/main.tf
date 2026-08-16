data "aws_iam_policy_document" "ecr" {
  statement {
    effect  = "Allow"
    actions = ["ecr:GetDownloadUrlForLayer", "ecr:BatchGetImage"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::111122223333:root"]
    }
  }
}

resource "aws_ecr_repository" "this" {
  name = "example-repo"
}

resource "aws_ecr_repository_policy" "pass" {
  repository = aws_ecr_repository.this.name
  policy     = data.aws_iam_policy_document.ecr.json
}
