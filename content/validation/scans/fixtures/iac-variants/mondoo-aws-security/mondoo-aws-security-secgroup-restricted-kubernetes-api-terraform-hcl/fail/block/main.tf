# Non-compliant: sensitive-port ingress via nested block.
resource "aws_security_group" "example" {
  name = "example"

  ingress {
    description      = "app port"
    from_port        = 6443
    to_port          = 6443
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["fd00::/8"]
  }
}
