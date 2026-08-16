data "aws_iam_policy_document" "bus" {
  statement {
    sid     = "AllowProductionAccount"
    effect  = "Allow"
    actions = ["events:PutEvents"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::111122223333:root"]
    }
  }
}

resource "aws_cloudwatch_event_bus" "this" {
  name = "app-bus"
}

resource "aws_cloudwatch_event_bus_policy" "pass" {
  event_bus_name = aws_cloudwatch_event_bus.this.name
  policy         = data.aws_iam_policy_document.bus.json
}
