# Compliant for the VNC check: the VNC rule (5900-5903) is restricted to a corporate
# CIDR. A separate HTTPS (443) rule is open to 0.0.0.0/0, which is fine for a web tier and
# must NOT flag the VNC check — this exercises the port-scoped block-style clause so an
# unrelated open ingress rule does not cause a false positive.
resource "aws_security_group" "web" {
  name = "web"

  ingress {
    from_port   = 5900
    to_port     = 5903
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
