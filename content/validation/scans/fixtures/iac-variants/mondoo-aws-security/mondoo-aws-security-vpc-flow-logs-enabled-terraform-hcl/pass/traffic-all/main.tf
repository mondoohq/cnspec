# Compliant: VPC has a flow log capturing ALL traffic.
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_flow_log" "main" {
  vpc_id          = aws_vpc.main.id
  traffic_type    = "ALL"
  log_destination = aws_cloudwatch_log_group.flow.arn
  iam_role_arn    = aws_iam_role.flow.arn
}

resource "aws_cloudwatch_log_group" "flow" {
  name = "vpc-flow-logs"
}

resource "aws_iam_role" "flow" {
  name               = "vpc-flow-log-role"
  assume_role_policy = "{}"
}
