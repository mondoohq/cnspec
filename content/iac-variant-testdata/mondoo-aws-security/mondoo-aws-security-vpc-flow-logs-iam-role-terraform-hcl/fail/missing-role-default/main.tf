# Non-compliant: CloudWatch Logs flow log (default type) without an IAM role.
resource "aws_flow_log" "default" {
  vpc_id          = aws_vpc.main.id
  traffic_type    = "ALL"
  log_destination = aws_cloudwatch_log_group.flow.arn
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_cloudwatch_log_group" "flow" {
  name = "vpc-flow-logs"
}
