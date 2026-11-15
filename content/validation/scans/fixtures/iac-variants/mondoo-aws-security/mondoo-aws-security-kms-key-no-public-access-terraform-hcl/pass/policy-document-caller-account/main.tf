# Compliant: aws_iam_policy_document wildcard principal scoped by kms:CallerAccount.
data "aws_iam_policy_document" "key" {
  statement {
    effect    = "Allow"
    actions   = ["kms:Decrypt"]
    resources = ["*"]

    principals {
      type        = "AWS"
      identifiers = ["*"]
    }

    condition {
      test     = "StringEquals"
      variable = "kms:CallerAccount"
      values   = ["123456789012"]
    }
  }
}

resource "aws_kms_key" "example" {
  description = "application data key"
  policy      = data.aws_iam_policy_document.key.json
}
