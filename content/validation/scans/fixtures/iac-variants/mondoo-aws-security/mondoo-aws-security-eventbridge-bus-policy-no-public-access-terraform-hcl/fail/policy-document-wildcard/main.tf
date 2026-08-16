data "aws_iam_policy_document" "bus" {
  statement {
    effect  = "Allow"
    actions = ["events:PutEvents"]

    principals {
      type        = "AWS"
      identifiers = ["*"]
    }
  }
}

resource "aws_cloudwatch_event_bus" "this" {
  name = "app-bus"
}

resource "aws_cloudwatch_event_bus_policy" "fail" {
  event_bus_name = aws_cloudwatch_event_bus.this.name
  policy         = data.aws_iam_policy_document.bus.json
}
