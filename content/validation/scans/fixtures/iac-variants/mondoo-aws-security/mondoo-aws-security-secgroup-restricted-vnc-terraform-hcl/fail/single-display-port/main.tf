# A rule opening only display :0 does not span 5900-5903. The check must still
# match it, so the port test has to be an overlap and not a containment.
resource "aws_security_group" "vnc" {
  name = "vnc"

  ingress {
    from_port   = 5900
    to_port     = 5900
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
