# Non-compliant: sensitive-port ingress via nested block.
resource "aws_security_group" "example" {
  name = "example"

  ingress {
    description      = "app port"
    from_port        = 10250
    to_port          = 10255
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["fd00::/8"]
  }
}
