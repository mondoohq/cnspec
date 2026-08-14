# monitor mode reports unencrypted flows but does not block them, so traffic
# still crosses the VPC in cleartext.
resource "aws_vpc" "prod" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_vpc_encryption_control" "prod" {
  vpc_id = aws_vpc.prod.id
  mode   = "monitor"
}
