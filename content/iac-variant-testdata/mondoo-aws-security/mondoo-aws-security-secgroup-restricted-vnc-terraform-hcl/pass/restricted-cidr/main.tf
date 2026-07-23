resource "aws_security_group" "vnc" {
  name = "vnc"

  ingress {
    from_port   = 5900
    to_port     = 5903
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }
}
