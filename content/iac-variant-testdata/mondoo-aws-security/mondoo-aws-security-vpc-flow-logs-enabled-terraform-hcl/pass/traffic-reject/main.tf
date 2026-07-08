# Compliant: VPC has a flow log capturing REJECT traffic.
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_flow_log" "reject" {
  vpc_id                   = aws_vpc.main.id
  traffic_type             = "REJECT"
  log_destination_type     = "s3"
  log_destination          = aws_s3_bucket.flow.arn
  max_aggregation_interval = 60
}

resource "aws_s3_bucket" "flow" {
  bucket = "vpc-flow-logs-example"
}
