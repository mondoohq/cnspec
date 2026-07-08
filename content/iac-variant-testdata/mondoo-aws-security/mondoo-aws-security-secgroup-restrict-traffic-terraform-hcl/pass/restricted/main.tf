# Compliant: an all-ports ingress rule (to_port -1) restricted to a private CIDR.
resource "aws_security_group" "pass_example" {
  name = "example"

  ingress {
    from_port   = 0
    to_port     = -1
    protocol    = "-1"
    cidr_blocks = ["10.0.0.0/8"]
  }
}
