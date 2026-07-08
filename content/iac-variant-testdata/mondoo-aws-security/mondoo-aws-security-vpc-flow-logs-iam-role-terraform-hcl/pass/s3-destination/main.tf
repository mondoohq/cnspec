# Compliant: S3 flow logs do not require an IAM role (delivery is direct to S3).
resource "aws_flow_log" "s3" {
  vpc_id               = aws_vpc.main.id
  traffic_type         = "ALL"
  log_destination_type = "s3"
  log_destination      = aws_s3_bucket.flow.arn
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_s3_bucket" "flow" {
  bucket = "vpc-flow-logs-example"
}
