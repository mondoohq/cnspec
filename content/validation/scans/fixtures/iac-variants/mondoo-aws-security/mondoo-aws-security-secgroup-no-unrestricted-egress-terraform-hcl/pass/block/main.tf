# Compliant: egress via nested block.
resource "aws_security_group" "example" {
  name = "example"

  egress {
    description      = "outbound"
    from_port        = 443
    to_port          = 443
    protocol         = "tcp"
    cidr_blocks      = ["10.0.0.0/8"]
    ipv6_cidr_blocks = ["fd00::/8"]
  }
}
