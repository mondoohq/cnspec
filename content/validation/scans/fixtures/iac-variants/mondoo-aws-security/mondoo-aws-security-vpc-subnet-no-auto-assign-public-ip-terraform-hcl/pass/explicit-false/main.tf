# Compliant: subnet explicitly disables auto-assign public IP.
resource "aws_subnet" "private" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = false
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}
