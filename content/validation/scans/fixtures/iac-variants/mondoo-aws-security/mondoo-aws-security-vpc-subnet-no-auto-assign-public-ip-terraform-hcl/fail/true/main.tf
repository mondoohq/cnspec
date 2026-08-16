# Non-compliant: subnet auto-assigns public IPs on instance launch.
resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.3.0/24"
  map_public_ip_on_launch = true
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}
