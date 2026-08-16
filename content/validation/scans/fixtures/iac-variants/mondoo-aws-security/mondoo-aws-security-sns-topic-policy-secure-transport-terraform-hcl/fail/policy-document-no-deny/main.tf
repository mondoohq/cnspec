data "aws_iam_policy_document" "publish" {
  statement {
    effect  = "Allow"
    actions = ["sns:Publish"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::111122223333:root"]
    }
  }
}

resource "aws_sns_topic" "this" {
  name = "app-topic"
}

resource "aws_sns_topic_policy" "fail" {
  arn    = aws_sns_topic.this.arn
  policy = data.aws_iam_policy_document.publish.json
}
