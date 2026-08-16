data "aws_iam_policy_document" "secure_transport" {
  statement {
    sid     = "DenyInsecureTransport"
    effect  = "Deny"
    actions = ["sns:*"]

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }
}

resource "aws_sns_topic" "this" {
  name = "app-topic"
}

resource "aws_sns_topic_policy" "pass" {
  arn    = aws_sns_topic.this.arn
  policy = data.aws_iam_policy_document.secure_transport.json
}
