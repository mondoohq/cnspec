# Non-compliant: sensitive-port ingress via nested block.
resource "aws_security_group" "example" {
  name = "example"

  ingress {
    description      = "app port"
    from_port        = 2379
    to_port          = 2380
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["fd00::/8"]
  }
}
