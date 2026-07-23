# Non-compliant: RDP (3389) ingress open to the world (block form).
resource "aws_security_group" "rdp" {
  name        = "rdp"
  description = "open rdp access"

  ingress {
    from_port   = 3389
    to_port     = 3389
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
