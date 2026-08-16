# Compliant: attribute omitted, so it defaults to false.
resource "aws_subnet" "private" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = "us-east-1a"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}
