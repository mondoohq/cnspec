data "aws_iam_policy_document" "efs" {
  statement {
    effect  = "Allow"
    actions = ["elasticfilesystem:ClientMount"]

    principals {
      type        = "*"
      identifiers = ["*"]
    }
  }
}

resource "aws_efs_file_system" "this" {
  creation_token = "app-data"
}

resource "aws_efs_file_system_policy" "fail" {
  file_system_id = aws_efs_file_system.this.id
  policy         = data.aws_iam_policy_document.efs.json
}
