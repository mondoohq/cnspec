data "aws_iam_policy_document" "efs" {
  statement {
    effect  = "Allow"
    actions = ["elasticfilesystem:ClientMount", "elasticfilesystem:ClientWrite"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::111122223333:role/AppRole"]
    }
  }
}

resource "aws_efs_file_system" "this" {
  creation_token = "app-data"
}

resource "aws_efs_file_system_policy" "pass" {
  file_system_id = aws_efs_file_system.this.id
  policy         = data.aws_iam_policy_document.efs.json
}
