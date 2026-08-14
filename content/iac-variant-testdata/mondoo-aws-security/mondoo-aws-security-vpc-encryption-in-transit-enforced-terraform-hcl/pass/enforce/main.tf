resource "aws_vpc" "prod" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
}

resource "aws_vpc_encryption_control" "prod" {
  vpc_id = aws_vpc.prod.id
  mode   = "enforce"
}
