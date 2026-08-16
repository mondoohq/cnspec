# Non-compliant: explicit cloud-watch-logs destination with an empty IAM role.
resource "aws_flow_log" "cwl" {
  vpc_id               = aws_vpc.main.id
  traffic_type         = "ALL"
  log_destination_type = "cloud-watch-logs"
  log_destination      = aws_cloudwatch_log_group.flow.arn
  iam_role_arn         = ""
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_cloudwatch_log_group" "flow" {
  name = "vpc-flow-logs"
}
