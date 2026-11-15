# Non-compliant: aws_iam_policy_document wildcard principal with a condition that does not limit callers.
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
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["true"]
    }
  }
}

resource "aws_kms_key" "example" {
  description = "application data key"
  policy      = data.aws_iam_policy_document.key.json
}
